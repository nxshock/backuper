package main

// FileHistory содержит историю изменения файла
type FileHistory []FileInfo

// Latest возвращает информацию о последней версии файла
func (fileHistory FileHistory) Latest() FileInfo {
	file := fileHistory[0]

	for i := 1; i < len(fileHistory); i++ {
		if fileHistory[i].ModificationTime.After(file.ModificationTime) {
			file = fileHistory[i]
		}
	}
	return file
}

func (fileHistory FileHistory) Len() int {
	return len(fileHistory)
}

func (fileHistory FileHistory) Swap(i, j int) {
	fileHistory[i], fileHistory[j] = fileHistory[j], fileHistory[i]
}

func (fileHistory FileHistory) Less(i, j int) bool {
	return fileHistory[i].ModificationTime.Before(fileHistory[j].ModificationTime)
}
