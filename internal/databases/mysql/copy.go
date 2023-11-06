package mysql

import (
	"path"
	"strings"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/copy"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

// HandleCopyBackup copy specific backups from one storage to another
func HandleCopyBackup(fromConfigFile, toConfigFile, backupName, prefix string) {
	var from, fromError = internal.FolderFromConfig(fromConfigFile)
	var to, toError = internal.FolderFromConfig(toConfigFile)
	if fromError != nil || toError != nil {
		return
	}
	infos, err := backupCopyingInfo(backupName, prefix, from, to)
	tracelog.ErrorLogger.FatalOnError(err)

	tracelog.DebugLogger.Printf("copying files %s\n", strings.Join(func() []string {
		ret := make([]string, 0)
		for _, e := range infos {
			ret = append(ret, e.SrcObj.GetName())
		}

		return ret
	}(), ","))

	tracelog.ErrorLogger.FatalOnError(copy.Infos(infos))

	tracelog.InfoLogger.Printf("Success copyed backup %s.\n", backupName)
}

// HandleCopyBackup copy  all backups from one storage to another
func HandleCopyAll(fromConfigFile string, toConfigFile string) {
	var from, fromError = internal.FolderFromConfig(fromConfigFile)
	var to, toError = internal.FolderFromConfig(toConfigFile)
	if fromError != nil || toError != nil {
		return
	}
	infos, err := WildcardInfo(from, to)
	tracelog.ErrorLogger.FatalOnError(err)
	err = copy.Infos(infos)
	tracelog.ErrorLogger.FatalOnError(err)
	tracelog.InfoLogger.Printf("Success copyed all backups\n")
}

func backupCopyingInfo(backupName, prefix string, from storage.Folder, to storage.Folder) ([]copy.InfoProvider, error) {
	tracelog.InfoLogger.Printf("Handle backupname '%s'.", backupName)
	backup, err := internal.GetBackupByName(backupName, utility.BaseBackupPath, from)
	if err != nil {
		return nil, err
	}
	tracelog.InfoLogger.Print("Collecting backup files...")
	var backupPrefix = path.Join(utility.BaseBackupPath, backup.Name)

	objects, err := storage.ListFolderRecursively(from)
	if err != nil {
		return nil, err
	}

	var hasBackupPrefix = func(object storage.Object) bool { return strings.HasPrefix(object.GetName(), backupPrefix) }
	return copy.BuildCopyingInfos(from, to, objects, hasBackupPrefix, func(object storage.Object) string {
		return strings.Replace(object.GetName(), backup.Name, prefix+backup.Name, 1)
	}), nil
}

func WildcardInfo(from storage.Folder, to storage.Folder) ([]copy.InfoProvider, error) {
	objects, err := storage.ListFolderRecursively(from)
	if err != nil {
		return nil, err
	}

	return copy.BuildCopyingInfos(from, to, objects, func(object storage.Object) bool { return true },
		copy.NoopRenameFunc), nil
}
