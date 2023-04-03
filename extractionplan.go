package main

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
)

type ExtractionPlan map[string][]string // filepath - array of internal paths

func (b *Config) extractionPlan(mask string, t time.Time) (ExtractionPlan, error) {
	index, err := b.index(true)
	if err != nil {
		return nil, fmt.Errorf("extractionPlan: %v", err)
	}

	files, err := index.GetFilesLocation(mask, t)
	if err != nil {
		return nil, fmt.Errorf("extractionPlan: %v", err)
	}

	plan := make(ExtractionPlan)

	for _, file := range files {
		plan[file.ArchiveFileName] = append(plan[file.ArchiveFileName], file.filePath)
	}

	return plan, nil
}

func (b *Config) extract(extractionPlan ExtractionPlan, toDir string) error {
	for archiveFile, files := range extractionPlan {
		log.Printf("Восстановление из архивного файла %s...", filepath.Join(filepath.Dir(b.filePath), archiveFile))
		f, err := os.Open(filepath.Join(filepath.Dir(b.filePath), archiveFile))
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
			if inArr, i := stringIn(header.Name, files); inArr {
				log.Printf("Восстановление файла %s...", header.Name)
				resultFilePath := filepath.Join(toDir, clean(header.Name))
				os.MkdirAll(filepath.Dir(resultFilePath), 0644)
				f, err := os.Create(resultFilePath)
				if err != nil {
					return err
				}

				_, err = io.Copy(f, tarReader)
				if err != nil {
					f.Close() // TODO: удалять частичный файл?
					return fmt.Errorf("ошибка при извлечении файла из tar-архива: %v", err)
				}

				f.Close()
				if err != nil {
					return err
				}

				files[i] = files[len(files)-1]
				files = files[:len(files)-1]
			}
		}
	}
	return nil
}
