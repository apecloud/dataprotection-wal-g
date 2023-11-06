package internal

import (
	"io"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
)

type StorageFolderReader interface {
	ReadObject(objectRelativePath string) (io.ReadCloser, error)
	SubFolder(subFolderRelativePath string) StorageFolderReader
}

func NewFolderReader(folder storage.Folder) StorageFolderReader {
	return &FolderReaderImpl{folder}
}

type FolderReaderImpl struct {
	storage.Folder
}

func (fsr *FolderReaderImpl) SubFolder(subFolderRelativePath string) StorageFolderReader {
	return NewFolderReader(fsr.GetSubFolder(subFolderRelativePath))
}
