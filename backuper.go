package main

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
)

func (b *Config) fileList(fileNames chan FileInfo) {
	errorCount := 0

	for _, mask := range b.Patterns {
		if mask.Recursive {
			err := filepath.WalkDir(mask.Path, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					errorCount++
					b.logf(Error, "Ошибка при поиске файлов: %v\n", err)
					if b.StopOnAnyError {
						return fmt.Errorf("ошибка при переборе файлов: %v", err)
					}
					return nil
				}

				if d.IsDir() {
					return nil
				}

				path = filepath.ToSlash(path)

				if !mask.Recursive && filepath.Dir(path) != mask.Path {
					return nil
				}

				if isFilePathMatchPatterns(mask.FilePathPatternList, path) && isFileNameMatchPatterns(mask.FileNamePatternList, path) {
					if !isFilePathMatchPatterns(b.GlobalExcludeFilePathPatterns, path) && !isFileNameMatchPatterns(b.GlobalExcludeFileNamePatterns, path) {
						info, err := os.Stat(path)
						if err != nil {
							errorCount++
							b.logf(Error, "get file info error: %v", err)
							if b.StopOnAnyError {
								return fmt.Errorf("get file info error: %v", err)
							}
						}

						file := FileInfo{
							filePath:         path,
							ModificationTime: info.ModTime(),
							fileSize:         info.Size()}
						fileNames <- file
					}
				}
				return nil
			})
			if err != nil {
				b.logf(Error, "get file list error: %v\n", err)
			}
		} else {
			allFilesAndDirs, err := filepath.Glob(filepath.Join(mask.Path, "*"))
			if err != nil {
				errorCount++
				b.logf(Error, "get file list error: %v\n", err)
			}

			for _, fileOrDirPath := range allFilesAndDirs {
				info, err := os.Stat(fileOrDirPath)
				if err != nil {
					errorCount++
					b.logf(Error, "get object info error: %v\n", err)
					continue
				}

				if info.IsDir() {
					continue
				}

				if isFilePathMatchPatterns(mask.FilePathPatternList, fileOrDirPath) && isFileNameMatchPatterns(mask.FileNamePatternList, fileOrDirPath) {
					if !isFilePathMatchPatterns(b.GlobalExcludeFilePathPatterns, fileOrDirPath) && !isFileNameMatchPatterns(b.GlobalExcludeFileNamePatterns, fileOrDirPath) {
						file := FileInfo{
							filePath:         fileOrDirPath,
							ModificationTime: info.ModTime()}
						fileNames <- file
					}
				}
			}
		}
	}

	if errorCount > 0 {
		b.logf(Error, "Ошибок: %d\n", errorCount)
	}

	close(fileNames)
}

func (b *Config) FullBackup() error {
	return b.doBackup(make(Index))
}

func (b *Config) IncrementalBackup() error {
	index, err := b.index(false)
	if err != nil {
		return err
	}

	return b.doBackup(index)
}

func (b *Config) doBackup(index Index) error {
	var suffix string
	if len(index) == 0 {
		suffix = "f" // Full backup - полный бекап
	} else {
		suffix = "i" // Инкрементальный бекап
	}

	filePath := filepath.Join(filepath.Dir(b.filePath), b.FileName+"_"+time.Now().Local().Format(defaulFileNameTimeFormat)+suffix+defaultExt)

	var err error
	filePath, err = filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("ошибка при создании файла архива: %v", err)
	}
	b.logf(Info, "Creating new file %s...", filepath.Base(filePath))

	resultArchiveFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("ошибка при создании файла архива: %v", err)
	}

	compressor, err := zstd.NewWriter(resultArchiveFile, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return fmt.Errorf("ошибка при создании инициализации архиватора: %v", err)
	}

	tarWriter := tar.NewWriter(compressor)

	b.log(Info, "Copying files...")

	addedFileIndex := make(Index)

	i := 0              // processed file count
	addSize := int64(0) // added bytes
	for k := range b.planChan(index) {
		i++
		addSize += k.fileSize
		err := b.addFileToTarWriter(k.filePath, tarWriter)
		if err != nil {
			b.logf(Error, "add file error %s: %v\n", k.filePath, err)
			if b.StopOnAnyError {
				compressor.Close()
				resultArchiveFile.Close()
				os.Remove(filePath)
				return fmt.Errorf("add file error: %v", err) // TODO: организовать закрытие и удаление частичного файла
			}
		}
		addedFileIndex.AddFile(k.filePath, filepath.Base(filePath), k.ModificationTime)
	}

	err = tarWriter.Close()
	if err != nil {
		compressor.Close()
		resultArchiveFile.Close()
		os.Remove(filePath)
		return fmt.Errorf("close tar file error: %v", err)
	}

	err = compressor.Close()
	if err != nil {
		resultArchiveFile.Close()
		os.Remove(filePath)
		return fmt.Errorf("close compressor error: %v", err)
	}

	err = resultArchiveFile.Close()
	if err != nil {
		return fmt.Errorf("close file error: %v", err)
	}

	if i == 0 {
		b.logf(Info, "No new or updated files found.")
	} else if i == 1 {
		b.logf(Info, "%d file added, %s.", i, sizeToApproxHuman(addSize))
	} else {
		b.logf(Info, "%d files added, %s.", i, sizeToApproxHuman(addSize))
	}

	// если не было обновлений, удалить пустой файл
	if i == 0 {
		err = os.Remove(filePath)
		if err != nil {
			return err
		}
	}

	// если были обновления - обновить индексный файл
	if i > 0 {
		for fileName, fileHistory := range addedFileIndex {
			for _, historyItem := range fileHistory {
				index.AddFile(fileName, historyItem.ArchiveFileName, historyItem.ModificationTime)
			}
		}

		err = index.Save(filepath.Join(filepath.Dir(b.filePath), indexFileName))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Config) addFileToTarWriter(filePath string, tarWriter *tar.Writer) error {
	b.logf(Debug, "Adding file %s...\n", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Could not open file '%s', got error '%s'", filePath, err.Error())
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Could not get stat for file '%s', got error '%s'", filePath, err.Error())
	}

	header := &tar.Header{
		Format:  tar.FormatGNU,
		Name:    filepath.ToSlash(filePath),
		Size:    stat.Size(),
		ModTime: stat.ModTime()}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("Could not write header for file '%s', got error '%s'", filePath, err.Error())
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return fmt.Errorf("Could not copy the file '%s' data to the tarball, got error '%s'", filePath, err.Error())
	}

	return nil
}
