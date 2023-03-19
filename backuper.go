package main

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/nxshock/progressmessage"
)

type Mask struct {
	Path string

	// Маски имени файла
	FileNameMaskList []string

	// Маски пути
	FilePathMaskList []string

	// Вкючать файлы в покаталогах
	Recursive bool
}

// FindAll возвращает индекс файлов, совпавших по маске
func (b *Config) FindAll(mask string) (*Index, error) {
	b.logf(LogLevelDebug, "Поиск маски %s...", mask)
	index, err := b.index(true)
	if err != nil {
		return nil, fmt.Errorf("index: %v", err)
	}

	result := &Index{Files: make(map[string]FileHistory)}

	for path, info := range index.Files {
		matched, err := filepath.Match(strings.ToLower(mask), strings.ToLower(filepath.ToSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("filepath.Match: %v", err)
		}
		if matched {
			result.Files[path] = append(result.Files[path], info...)
		}
	}

	return result, nil
}

// IncrementalBackup выполняет инкрементальный бекап.
// В случае, если бекап выполняется впервые, выполняется полный бекап.
func (b *Config) IncrementalBackup() error {
	index, err := b.index(false)
	if err != nil {
		return err
	}

	return b.doBackup(index)
}

// FullBackup выполняет полное резервное копирование
func (b *Config) FullBackup() error {
	return b.doBackup(nil)
}

func (b *Config) doBackup(index *Index) error {
	var suffix string
	if index == nil || index.ItemCount() == 0 {
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
	b.logf(LogLevelProgress, "Создание нового файла бекапа %s...", filePath)

	if _, err = os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(filePath), 0644)
		if err != nil {
			return fmt.Errorf("ошибка при создании каталога для архива: %v", err)
		}
	}

	resultArchiveFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("ошибка при создании файла архива: %v", err)
	}

	compressor, err := zstd.NewWriter(resultArchiveFile, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return fmt.Errorf("ошибка при создании инициализации архиватора: %v", err)
	}

	tarWriter := tar.NewWriter(compressor)

	b.log(LogLevelInfo, "Копирование файлов...")

	pm := progressmessage.New("Добавлено %d файлов, %s...")
	if b.Logger.MinimalLogLevel <= LogLevelProgress {
		pm.Start()
	}

	i := 0              // счётчик обработанных файлов
	addSize := int64(0) // добавлено байт
	for k := range b.planChan(index) {
		i++
		addSize += k.Info.Size()
		err := b.addFileToTarWriter(k.SourcePath, tarWriter)
		if err != nil {
			b.logf(LogLevelWarning, "ошибка при добавлении файла %s: %v\n", k.SourcePath, err)
			if b.StopOnAnyError {
				compressor.Close()
				resultArchiveFile.Close()
				os.Remove(filePath)
				return fmt.Errorf("ошибка при добавлении файла в архив: %v", err) // TODO: организовать закрытие и удаление частичного файла
			}
		}

		if b.Logger.MinimalLogLevel <= LogLevelProgress {
			pm.Update(i, sizeToApproxHuman(addSize))
		}
	}

	if b.Logger.MinimalLogLevel <= LogLevelProgress {
		pm.Stop()
	}

	err = tarWriter.Close()
	if err != nil {
		compressor.Close()
		resultArchiveFile.Close()
		os.Remove(filePath)
		return fmt.Errorf("ошибка при закрытии tar-архива: %v", err)
	}

	err = compressor.Close()
	if err != nil {
		resultArchiveFile.Close()
		os.Remove(filePath)
		return fmt.Errorf("ошибка при закрытии архива: %v", err)
	}

	if b.Logger.MinimalLogLevel <= LogLevelProgress {
		fmt.Fprintf(os.Stderr, "\rДобавлено %d файлов, %s.\n", i, sizeToApproxHuman(addSize))
	}

	err = resultArchiveFile.Close()
	if err != nil {
		return fmt.Errorf("ошибка при закрытии файла архива: %v", err)
	}

	// если не было обновлений, удалить пустой файл
	if i == 0 {
		os.Remove(filePath)
	}

	return nil
}

func (b *Config) fileList(fileNames chan File) {
	errorCount := 0

	for _, mask := range b.Masks {
		if mask.Recursive {
			err := filepath.WalkDir(mask.Path, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					errorCount++
					b.logf(LogLevelCritical, "Ошибка при поиске файлов: %v\n", err)
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

				if isFilePathMatchMasks(mask.FilePathMaskList, path) && isFileNameMatchMasks(mask.FileNameMaskList, path) {
					if !isFilePathMatchMasks(b.GlobalExcludeFilePathMasks, path) && !isFileNameMatchMasks(b.GlobalExcludeFileNameMasks, path) {
						info, err := os.Stat(path)
						if err != nil {
							errorCount++
							b.logf(LogLevelCritical, "Ошибка при получении информации о файле: %v\n", err)
							if b.StopOnAnyError {
								return fmt.Errorf("ошибка при получении информации о файле: %v", err)
							}
						}

						file := File{
							SourcePath:      path,
							DestinationPath: filepath.ToSlash(path),
							Info:            info}
						fileNames <- file
					}
				}
				return nil
			})
			if err != nil {
				b.logf(LogLevelCritical, "Ошибка при получении списка файлов: %v\n", err)
			}
		} else {
			allFilesAndDirs, err := filepath.Glob(filepath.Join(mask.Path, "*"))
			if err != nil {
				errorCount++
				b.logf(LogLevelCritical, "Ошибка при получении списка файлов: %v\n", err)
			}

			for _, fileOrDirPath := range allFilesAndDirs {
				info, err := os.Stat(fileOrDirPath)
				if err != nil {
					errorCount++
					b.logf(LogLevelCritical, "Ошибка при получении информации об объекте: %v\n", err)
					continue
				}

				if info.IsDir() {
					continue
				}

				//fileName := filepath.Base(fileOrDirPath)
				fileName := fileOrDirPath // TODO: тестирование, маска должна накладываться на путь

				if isFilePathMatchMasks(mask.FilePathMaskList, fileName) && isFileNameMatchMasks(mask.FileNameMaskList, fileName) {
					if !isFilePathMatchMasks(b.GlobalExcludeFilePathMasks, fileName) && !isFileNameMatchMasks(b.GlobalExcludeFileNameMasks, fileName) {
						file := File{
							SourcePath:      fileOrDirPath,
							DestinationPath: filepath.ToSlash(fileOrDirPath),
							Info:            info}
						fileNames <- file
					}
				}
			}
		}
	}

	if errorCount > 0 {
		b.logf(LogLevelCritical, "Ошибок: %d\n", errorCount)
	}

	close(fileNames)
}

func (b *Config) addFileToTarWriter(filePath string, tarWriter *tar.Writer) error {
	b.logf(LogLevelDebug, "Добавление файла %s...\n", filePath)

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

// GetFileWithTime возвращает содержимое файла на указанную дату.
func (b *Config) GetFileWithTime(path string, t time.Time, w io.Writer) error {
	index, err := b.index(true)
	if err != nil {
		return fmt.Errorf("ошибка при построении индекса: %v", err)
	}

	file, err := index.GetFileWithTime(path, t)
	if err != nil {
		return fmt.Errorf("ошибка при получении информации из индекса: %v", err)
	}

	f, err := os.Open(file.ArchiveFile)
	if err != nil {
		return fmt.Errorf("ошибка при чтении файла архива: %v", err)
	}
	defer f.Close()

	decoder, err := zstd.NewReader(f)
	if err != nil {
		return fmt.Errorf("ошибка при инициализации разархиватора: %v", err)
	}
	defer decoder.Close()

	tarReader := tar.NewReader(decoder)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ошибка при чтении tar-содержимого: %v", err)
		}
		if header.Name == path {
			_, err = io.Copy(w, tarReader)
			if err != nil {
				return fmt.Errorf("ошибка при извлечении файла из tar-архива: %v", err)
			}
			return nil
		}
	}

	return nil
}

// fullIndex - true = все файлы, false = только от последнего полного
func (b *Config) index(fullIndex bool) (*Index, error) {
	b.logf(LogLevelInfo, "Построение индекса текущего архива из %s...", filepath.Dir(b.filePath))
	allFileMask := filepath.Join(filepath.Dir(b.filePath), b.FileName+"*"+defaultExt)
	onlyFullBackupFileMask := filepath.Join(filepath.Dir(b.filePath), b.FileName+"*f"+defaultExt)

	// Get last full backup name
	lastFullBackupFileName := ""
	err := filepath.WalkDir(filepath.Dir(b.filePath), func(path string, info os.DirEntry, err error) error {
		matched, err := filepath.Match(onlyFullBackupFileMask, path)
		if err != nil {
			return fmt.Errorf("filepath.WalkDir: %v", err)
		}
		if !matched {
			return nil
		}

		lastFullBackupFileName = path

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("filepath.WalkDir: %v", err)
	}

	if !fullIndex {
		b.logf(LogLevelDebug, "Отсчёт производится от %s.", filepath.Base(lastFullBackupFileName))
	}

	var files []string
	err = filepath.WalkDir(filepath.Dir(b.filePath), func(path string, info os.DirEntry, err error) error {
		matched, err := filepath.Match(allFileMask, path)
		if err != nil {
			return fmt.Errorf("filepath.Match: %v", err)
		}
		if matched && path >= lastFullBackupFileName {
			if fullIndex || path >= lastFullBackupFileName {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("filepath.Walk: %v", err)
	}

	index := &Index{Files: make(map[string]FileHistory)}

	for i, file := range files {
		if b.logger.MinimalLogLevel <= LogLevelProgress {
			fmt.Fprintf(os.Stderr, "\r[%d%%] Чтение файла %s...", (100 * i / len(files)), filepath.Base(file))
		}
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("os.Open: %v", err)
		}
		defer f.Close()

		decoder, err := zstd.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("zstd.NewReader: %v", err)
		}

		tarReader := tar.NewReader(decoder)

		for {
			tarHeader, err := tarReader.Next()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, fmt.Errorf("ошибка при чтении списка файлов из архива %s: %v", file, err)
				}
			}

			b.logf(LogLevelDebug, "Найден файл %s...\n", tarHeader.Name)

			index.Files[tarHeader.Name] = append(index.Files[tarHeader.Name], File{
				DestinationPath: tarHeader.Name,
				Info:            tarHeader.FileInfo(),
				ArchiveFile:     file})
		}
		decoder.Close()
	}
	if b.logger.MinimalLogLevel <= LogLevelProgress && len(files) > 0 {
		fmt.Fprintf(os.Stderr, "\r[%d%%] Чтение файлов завершено.\n", 100) // TODO: нужна очистка строки, т.к. данная строка короче имени файлов
	}

	return index, nil
}

// Test осуществляет проверку архивов и возвращает первую встретившуюся ошибку
func (b *Config) Test() error {
	_, err := b.index(true) // TODO: улучшить реализацию

	return err
}
