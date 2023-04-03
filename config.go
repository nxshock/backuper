package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/match"

	"github.com/BurntSushi/toml"
)

type Config struct {
	// Имя файлов бекапа без расширения
	FileName string

	// Маски файлов для включения в архив
	Patterns []*Pattern

	// Маски файлов для исключения
	GlobalExcludeFileNamePatterns []string

	// Маски путей для исключения
	GlobalExcludeFilePathPatterns []string

	// Останавливать обработку при любой ошибке
	StopOnAnyError bool

	// Уровень логирования
	LogLevel LogLevel

	filePath string
}

func (config *Config) Save(filepath string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}

	err = toml.NewEncoder(f).Encode(config)
	if err != nil {
		f.Close()
		return err
	}

	return f.Close()
}

func LoadConfig(filePath string) (*Config, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %v", err)
	}
	defer f.Close()

	var config Config

	_, err = toml.DecodeReader(f, &config)
	if err != nil {
		return nil, fmt.Errorf("decode file: %v", err)
	}

	for _, mask := range config.Patterns {
		if len(mask.FilePathPatternList) == 0 {
			mask.FilePathPatternList = []string{"*"}
		}
	}

	configFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	config.filePath = configFilePath

	return &config, nil
}

// planChan возвращает канал, в который засылает список файлов для добавления/обновления
func (b *Config) planChan(index Index) chan FileInfo {
	allFilesChan := make(chan FileInfo, 64) // TODO: размер очереди?
	addFilesChan := make(chan FileInfo, 64) // TODO: размер очереди?

	go func() { b.fileList(allFilesChan) }()

	go func() {
		for file := range allFilesChan {
			// Если индекса нет, добавляются все файлы
			if index == nil {
				addFilesChan <- file
				continue
			}

			existingFile, exists := index[file.filePath]
			if !exists {
				addFilesChan <- file
				continue
			}

			if file.ModificationTime.Truncate(time.Second).After(existingFile.Latest().ModificationTime.Truncate(time.Second)) {
				addFilesChan <- file
				continue
			}

		}
		close(addFilesChan)
	}()

	return addFilesChan
}

// FindAll возвращает индекс файлов, совпавших по маске
func (b *Config) FindAll(pattern string) (Index, error) {
	index, err := b.index(true)
	if err != nil {
		return nil, fmt.Errorf("index: %v", err)
	}

	result := make(Index)

	for path, info := range index {
		if match.Match(strings.ToLower(path), pattern) {
			for _, historyItem := range info {
				result.AddFile(path, historyItem.ArchiveFileName, historyItem.ModificationTime)
			}
		}
	}

	return result, nil
}
