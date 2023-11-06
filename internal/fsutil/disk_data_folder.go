package fsutil

import (
	"io"
	"os"
	"path/filepath"
)

type DiskDataFolder struct {
	Path string
}

func NewDiskDataFolder(folderPath string) (*DiskDataFolder, error) {
	err := os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &DiskDataFolder{folderPath}, nil
}

func ExistingDiskDataFolder(folderPath string) (*DiskDataFolder, error) {
	return &DiskDataFolder{folderPath}, nil
}

func (folder *DiskDataFolder) OpenReadonlyFile(filename string) (io.ReadCloser, error) {
	filePath := filepath.Join(folder.Path, filename)
	file, err := os.Open(filePath)
	if err != nil && os.IsNotExist(err) {
		return file, NewNoSuchFileError(filename)
	}
	return file, err
}

func (folder *DiskDataFolder) OpenWriteOnlyFile(filename string) (io.WriteCloser, error) {
	filePath := filepath.Join(folder.Path, filename)
	return os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
}

func (folder *DiskDataFolder) CleanFolder() error {
	cleaner := FileSystemCleaner{}
	files, err := cleaner.GetFiles(folder.Path)
	if err != nil {
		return err
	}
	for _, file := range files {
		cleaner.Remove(filepath.Join(folder.Path, file))
	}
	return nil
}

func (folder *DiskDataFolder) FileExists(filename string) bool {
	filePath := filepath.Join(folder.Path, filename)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func (folder *DiskDataFolder) CreateFile(filename string) error {
	filePath := filepath.Join(folder.Path, filename)
	_, err := os.Create(filePath)

	return err
}

func (folder *DiskDataFolder) DeleteFile(filename string) error {
	filePath := filepath.Join(folder.Path, filename)
	return os.Remove(filePath)
}

func (folder *DiskDataFolder) RenameFile(oldFileName string, newFileName string) error {
	oldPath := filepath.Join(folder.Path, oldFileName)
	newPath := filepath.Join(folder.Path, newFileName)
	return os.Rename(oldPath, newPath)
}
