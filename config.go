package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Backuper struct {
	Config *Config
}

type Config struct {
	// Имя файлов бекапа без расширения
	FileName string

	// Маски файлов для включения в архив
	Masks []Mask

	// Маски файлов/путей для исключения из всех масок
	GlobalExcludeMasks []string

	// Логгер
	Logger LoggerConfig
	logger Logger

	// Останавливать обработку при любой ошибке
	StopOnAnyError bool

	filePath string
}

type LoggerConfig struct {
	Name            string
	MinimalLogLevel LogLevel
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

	config.logger = Logger{logger: log.New(os.Stderr, "", 0), MinimalLogLevel: config.Logger.MinimalLogLevel}

	configFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	config.filePath = configFilePath

	return &config, nil
}

// planChan возвращает канал, в который засылает список файлов для добавления/обновления
func (b *Config) planChan(index *Index) chan File {
	allFilesChan := make(chan File, 64) // TODO: размер очереди?
	addFilesChan := make(chan File, 64) // TODO: размер очереди?

	go func() { b.fileList(allFilesChan) }()

	go func() {
		for file := range allFilesChan {
			// Если индекса нет, добавляются все файлы
			if index == nil {
				addFilesChan <- file
				continue
			}

			existingFile, exists := index.Files[file.DestinationPath]
			if !exists {
				addFilesChan <- file
				continue
			}

			if file.Info.ModTime().Truncate(time.Second).After(existingFile.Latest().Info.ModTime().Truncate(time.Second)) {
				addFilesChan <- file
				continue
			}

		}
		close(addFilesChan)
	}()

	return addFilesChan
}
