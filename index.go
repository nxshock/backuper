package main

import (
	"archive/tar"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/klauspost/compress/zstd"
)

type FileInfo struct {
	ArchiveFileName  string
	ModificationTime time.Time

	filePath string
	fileSize int64
}

type Index map[string]FileHistory

func (index Index) AddFile(fileName string, archiveFileName string, modTime time.Time) {
	fileInfo := FileInfo{ArchiveFileName: archiveFileName, ModificationTime: modTime}

	if eFileInfo, exists := index[fileName]; exists {
		index[fileName] = append(eFileInfo, fileInfo)
		return
	}

	index[fileName] = FileHistory{fileInfo}
}

func (index Index) ViewFileVersions(w io.Writer) error {
	for filePath, fileHistory := range index {
		_, err := fmt.Fprintf(w, "%s\n", filePath)
		if err != nil {
			return err
		}

		for _, v := range fileHistory {
			_, err := fmt.Fprintf(w, "\t%s %s\n", v.ModificationTime.Format(defaultTimeFormat), v.ArchiveFileName)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func (index Index) Save(fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}

	enc, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		f.Close()
		os.Remove(fileName)
		return err
	}

	files := make([]string, 0, len(index))
	for fileName := range index {
		files = append(files, fileName)
	}

	// Sort file list for better compression
	sort.Strings(files)

	csvWriter := csv.NewWriter(enc)
	csvWriter.Comma = ';'

	for _, fileName := range files {
		for _, historyItem := range index[fileName] {
			err := csvWriter.Write([]string{fileName, historyItem.ArchiveFileName, strconv.Itoa(int(historyItem.ModificationTime.Unix()))})
			if err != nil {
				enc.Close()
				f.Close()
				os.Remove(fileName)
				return err
			}
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		enc.Close()
		f.Close()
		os.Remove(fileName)
		return err
	}

	err = enc.Close()
	if err != nil {
		f.Close()
		os.Remove(fileName)
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return err
}

func (b *Config) index(fullIndex bool) (Index, error) {
	index, err := b.indexFromFile()
	if err == nil {
		b.logf(Debug, "Index file contains %d of files.", len(index))
		return index, nil
	}
	b.logf(Error, "index file read error: %v", err)

	return b.indexFromDisk(fullIndex)
}

func (b *Config) indexFromFile() (Index, error) {
	index := make(Index)

	indexFileName := filepath.Join(filepath.Dir(b.filePath), indexFileName)

	f, err := os.Open(indexFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec, err := zstd.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer dec.Close()

	csvReader := csv.NewReader(dec)
	csvReader.Comma = ';'
	csvReader.FieldsPerRecord = 3
	for {
		data, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		unixTime, err := strconv.Atoi(data[2])
		if err != nil {
			return nil, err
		}

		index.AddFile(data[0], data[1], time.Unix(int64(unixTime), 0).Local())
	}

	return index, nil
}

func (b *Config) indexFromDisk(fullIndex bool) (Index, error) {
	b.logf(Info, "Rebuilding index from %s...", filepath.Dir(b.filePath))
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
		b.logf(Debug, "Diff will be calculated from %s.", filepath.Base(lastFullBackupFileName))
	}

	var files []string
	err = filepath.WalkDir(filepath.Dir(b.filePath), func(path string, info os.DirEntry, err error) error {
		matched, err := filepath.Match(allFileMask, path)
		if err != nil {
			return fmt.Errorf("filepath.Match: %v", err)
		}

		if matched && (fullIndex || path >= lastFullBackupFileName) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("filepath.Walk: %v", err)
	}

	index := make(Index)

	for i, file := range files {
		b.logf(Debug, "[%3d%%] Reading file %s...", (100 * i / len(files)), filepath.Base(file))
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

			index[tarHeader.Name] = append(index[tarHeader.Name], FileInfo{
				filePath:         tarHeader.Name,
				ModificationTime: tarHeader.FileInfo().ModTime(),
				fileSize:         tarHeader.FileInfo().Size(),
				ArchiveFileName:  filepath.Base(file)})
		}
		decoder.Close()
	}

	return index, nil
}

func (index Index) GetFilesLocation(mask string, t time.Time) ([]FileInfo, error) {
	var files2 []FileInfo

	for fileName := range index {
		if isFilePathMatchPatterns([]string{mask}, fileName) {
			files := index[fileName]

			file := files[0]
			for i := 1; i < len(files); i++ {
				if files[i].ModificationTime.Before(t) && files[i].ModificationTime.Sub(t) > file.ModificationTime.Sub(t) { // Больше, т.к. отрицательные числа
					file = files[i]
				}
			}

			file.filePath = fileName

			files2 = append(files2, file)
		}
	}

	return files2, nil
}
