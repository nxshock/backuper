package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIndexAddFile(t *testing.T) {
	index := make(Index)
	assert.Len(t, index, 0)

	fileName := "file"
	archiveFileName := "archive"
	modTime := time.Now()

	index.AddFile(fileName, archiveFileName, modTime)
	assert.Len(t, index, 1)
	assert.Len(t, index[fileName], 1)

	expectedFileInfo := FileInfo{
		ArchiveFileName:  archiveFileName,
		ModificationTime: modTime}

	assert.Equal(t, expectedFileInfo, index[fileName][0])
}
