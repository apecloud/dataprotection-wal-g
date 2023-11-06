package fs

import (
	"github.com/apecloud/datasafed/pkg/storage/rclone"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
)

const dirDefaultMode = 0755

func NewError(err error, format string, args ...interface{}) storage.Error {
	return storage.NewError(err, "datasafed", format, args...)
}

// Folder represents folder of file system
type Folder struct {
	rootPath string
	subpath  string
}

func NewFolder(rootPath string, subPath string) *Folder {
	rclone.New(map[string]string{})
	return &Folder{rootPath, subPath}
}
