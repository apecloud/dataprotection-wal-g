package fsutil

import (
	"os"

	"github.com/wal-g/tracelog"
)

// FileSystemCleaner actually performs it's functions on file system
type FileSystemCleaner struct{}

// GetFiles of a directory
func (cleaner FileSystemCleaner) GetFiles(directory string) (files []string, err error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return
	}
	files = make([]string, 0)
	for i := 0; i < len(entries); i++ {
		if entries[i].Type().IsDir() {
			continue
		}
		files = append(files, entries[i].Name())
	}
	return
}

// Remove file
func (cleaner FileSystemCleaner) Remove(file string) {
	err := os.Remove(file)
	if err != nil {
		tracelog.WarningLogger.Printf("Tried to remove file: '%s', but got error: '%v'\n", file, err)
	}
}
