package datasafed

import (
	"context"
	"errors"
	"io"
	"path"
	"strings"
	"sync"

	"github.com/apecloud/datasafed/pkg/app"
	ds "github.com/apecloud/datasafed/pkg/storage"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
)

const (
	defaultConfigFilePath = "/etc/datasafed/datasafed.conf"
)

var initOnce sync.Once

func NewError(err error, format string, args ...interface{}) storage.Error {
	return storage.NewError(err, "datasafed", format, args...)
}

// Folder represents folder of file system
type Folder struct {
	subPath string
	storage ds.Storage
	ctx     context.Context
}

func ConfigureFolder(configPath string, settings map[string]string) (storage.Folder, error) {
	return NewFolder(configPath, "")
}

func NewFolder(configFilePath string, subPath string) (*Folder, error) {
	if configFilePath == "" {
		configFilePath = defaultConfigFilePath
	}
	ctx := context.Background()
	var err error
	initOnce.Do(func() {
		err = app.InitGlobalStorage(ctx, configFilePath)
		if err == nil {
			cobra.OnFinalize(func() {
				app.InvokeFinalizers()
			})
		}
	})
	if err != nil {
		return nil, NewError(err, "init datasafed storage failed, config file: %s", configFilePath)
	}
	globalStorage, err := app.GetGlobalStorage()
	if err != nil {
		return nil, NewError(err, "failed to get global storage")
	}
	return &Folder{storage: globalStorage, subPath: subPath, ctx: ctx}, nil
}

// GetPath gets the root path.
func (folder *Folder) GetPath() string {
	return folder.subPath
}

func (folder *Folder) ListFolder() (objects []storage.Object, subFolders []storage.Folder, err error) {
	err = folder.storage.List(folder.ctx, folder.subPath, &ds.ListOptions{}, func(entry ds.DirEntry) error {
		if entry.IsDir() {
			// not using GetSubFolder() by intention
			subPath := path.Join(folder.subPath, entry.Name()) + "/"
			subFolders = append(subFolders, &Folder{
				ctx:     folder.ctx,
				storage: folder.storage,
				subPath: subPath,
			})
		} else {
			objects = append(objects, storage.NewLocalObject(entry.Name(), entry.MTime(), entry.Size()))
		}
		return nil
	})
	return
}

func (folder *Folder) DeleteObjects(objectRelativePaths []string) error {
	for _, fileName := range objectRelativePaths {
		filePath := folder.GetFilePath(fileName)
		err := folder.storage.Remove(folder.ctx, filePath, false)
		if err == nil || folder.isNotFoundError(err) {
			continue
		}
		// remove all for the dir
		err = folder.storage.Remove(folder.ctx, filePath, true)
		if folder.isNotFoundError(err) {
			continue
		}
		if err != nil {
			return NewError(err, "Unable to delete object %v", fileName)
		}
	}
	return nil
}

func (folder *Folder) Exists(objectRelativePath string) (bool, error) {
	_, err := folder.storage.Stat(folder.ctx, folder.GetFilePath(objectRelativePath))
	if folder.isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, NewError(err, "Unable to stat object %v", objectRelativePath)
	}
	return true, nil
}

func (folder *Folder) GetSubFolder(subFolderRelativePath string) storage.Folder {
	// This is something unusual when we cannot be sure that our subfolder exists in FS
	// But we do not have to guarantee folder persistence, but any subsequent calls will fail
	// Just like in all other Storage Folders
	subFolderPath := path.Join(folder.subPath, subFolderRelativePath)
	_, err := folder.storage.Stat(folder.ctx, subFolderPath)
	if err != nil {
		// make sure the dir exists
		_ = folder.storage.Mkdir(folder.ctx, subFolderPath)
	}
	return &Folder{
		ctx:     folder.ctx,
		subPath: subFolderPath,
		storage: folder.storage,
	}
}

func (folder *Folder) ReadObject(objectRelativePath string) (io.ReadCloser, error) {
	filePath := folder.GetFilePath(objectRelativePath)
	reader, err := folder.storage.OpenFile(folder.ctx, filePath, 0, -1)
	if err != nil {
		if errors.As(err, &ds.ErrObjectNotFound) {
			return reader, storage.NewObjectNotFoundError(objectRelativePath)
		}
		return reader, err
	}
	return reader, nil
}

func (folder *Folder) PutObject(name string, content io.Reader) error {
	tracelog.DebugLogger.Printf("Put %v into %v\n", name, folder.subPath)
	filePath := folder.GetFilePath(name)
	err := folder.storage.Push(folder.ctx, content, filePath)
	if err != nil {
		return NewError(err, "Unable to open file %v", filePath)
	}
	return nil
}

func (folder *Folder) CopyObject(srcPath string, dstPath string) error {
	readerCloser, err := folder.storage.OpenFile(folder.ctx, srcPath, 0, -1)
	if err != nil {
		return err
	}
	return folder.PutObject(dstPath, readerCloser)
}

func (folder *Folder) isNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

func (folder *Folder) GetFilePath(objectRelativePath string) string {
	return path.Join(folder.subPath, objectRelativePath)
}
