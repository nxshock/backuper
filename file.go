package main

import (
	"os"
)

type File struct {
	// Исходный путь
	SourcePath string

	// Путь в архиве
	DestinationPath string

	// Путь к архиву
	ArchiveFile string

	// Информация о файле
	Info os.FileInfo
}
