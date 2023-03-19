package main

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"time"
)

type Index struct {
	Files map[string]FileHistory // Путь -
}

func (fileHistory FileHistory) String() string {
	var b bytes.Buffer

	b.WriteString("[")
	for i := 0; i < len(fileHistory); i++ {
		if i > 0 {
			fmt.Fprintf(&b, ", %s", fileHistory[i].Info.ModTime().Local().Format(defaultTimeFormat))
		} else {
			fmt.Fprintf(&b, "%s", fileHistory[i].Info.ModTime().Local().Format(defaultTimeFormat))
		}
	}
	b.WriteString("]")

	return b.String()
}

func (index *Index) ItemCount() int {
	return len(index.Files)
}

func (index *Index) GetFileWithTime(path string, t time.Time) (File, error) {
	files, exists := index.Files[path]
	if !exists {
		return File{}, errors.New("not exists")
	}

	file := files[0]

	for i := 1; i < len(files); i++ {
		if files[i].Info.ModTime().Before(t) && files[i].Info.ModTime().Sub(t) > file.Info.ModTime().Sub(t) { // Больше, т.к. отрицательные числа
			file = files[i]
		}
	}

	return file, nil
}

func (index *Index) String() string {
	var b bytes.Buffer

	for path, info := range index.Files {
		sort.Sort(info)

		fmt.Fprintf(&b, "%s %s\n", path, info)
	}

	if b.Len() > 0 {
		b.Truncate(b.Len() - 1)
	}

	return b.String()
}

func (index *Index) GetFilesLocation(mask string, t time.Time) ([]File, error) {
	var files2 []File

	for fileName := range index.Files {
		if isFilePathMatchPatterns([]string{mask}, fileName) {
			files := index.Files[fileName]

			file := files[0]
			for i := 1; i < len(files); i++ {
				if files[i].Info.ModTime().Before(t) && files[i].Info.ModTime().Sub(t) > file.Info.ModTime().Sub(t) { // Больше, т.к. отрицательные числа
					file = files[i]
				}
			}

			files2 = append(files2, file)
		}
	}

	return files2, nil
}
