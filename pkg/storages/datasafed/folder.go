package datasafed

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	dfconfig "github.com/apecloud/datasafed/pkg/config"
	dfstorage "github.com/apecloud/datasafed/pkg/storage"
	"github.com/apecloud/datasafed/pkg/storage/rclone"
	"github.com/wal-g/tracelog"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
)

const (
	defaultConfigFilePath = "/etc/datasafed/datasafed.conf"
	backendBasePathEnv    = "DATASAFED_BACKEND_BASE_PATH"
	rootKey               = "root"
)

func NewError(err error, format string, args ...interface{}) storage.Error {
	return storage.NewError(err, "datasafed", format, args...)
}

// Folder represents folder of file system
type Folder struct {
	subPath string
	storage dfstorage.Storage
	ctx     context.Context
}

func ConfigureFolder(configPath string, settings map[string]string) (storage.Folder, error) {
	return NewFolder(configPath, "")
}

func NewFolder(configFilePath string, subPath string) (*Folder, error) {
	if configFilePath == "" {
		configFilePath = defaultConfigFilePath
	}
	if err := dfconfig.InitGlobal(configFilePath); err != nil {
		return nil, NewError(err, "init datasafed config failed, config file: %s", configFilePath)
	}
	storageConf := dfconfig.GetGlobal().GetAll(dfconfig.StorageSection)
	adjustRoot(storageConf)
	globalStorage, err := rclone.New(storageConf)
	if err != nil {
		return nil, err
	}
	return &Folder{storage: globalStorage, subPath: subPath, ctx: context.Background()}, nil
}

func adjustRoot(storageConf map[string]string) error {
	basePath := os.Getenv(backendBasePathEnv)
	if basePath == "" {
		return nil
	}

	basePath = filepath.Clean(basePath)
	if strings.HasPrefix(basePath, "..") {
		return fmt.Errorf("invalid base path %q from env %s",
			os.Getenv(backendBasePathEnv), backendBasePathEnv)
	}
	if basePath == "." {
		basePath = ""
	} else {
		basePath = strings.TrimPrefix(basePath, "/")
		basePath = strings.TrimPrefix(basePath, "./")
	}
	root := storageConf[rootKey]
	if strings.HasSuffix(root, "/") {
		root = root + basePath
	} else {
		root = root + "/" + basePath
	}
	path.Join()
	storageConf[rootKey] = root
	return nil
}

// GetPath gets the root path.
func (folder *Folder) GetPath() string {
	return folder.subPath
}

func (folder *Folder) ListFolder() (objects []storage.Object, subFolders []storage.Folder, err error) {
	entries, err := folder.storage.List(folder.ctx, folder.subPath, &dfstorage.ListOptions{})
	if err != nil {
		return nil, nil, NewError(err, "Unable to list folder")
	}
	for _, fileInfo := range entries {
		if fileInfo.IsDir() {
			// I do not use GetSubfolder() intentially
			subPath := path.Join(folder.subPath, fileInfo.Name()) + "/"
			subFolders = append(subFolders, &Folder{
				ctx:     folder.ctx,
				storage: folder.storage,
				subPath: subPath,
			})
		} else {
			objects = append(objects, storage.NewLocalObject(fileInfo.Name(), fileInfo.MTime(), fileInfo.Size()))
		}
	}
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
	return folder.storage.ReadObject(folder.ctx, filePath)
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
	readerCloser, err := folder.storage.ReadObject(folder.ctx, srcPath)
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
