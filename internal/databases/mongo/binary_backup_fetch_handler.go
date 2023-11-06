package mongo

import (
	"context"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/binary"
	"github.com/apecloud/dataprotection-wal-g/utility"
)

func HandleBinaryFetchPush(ctx context.Context, mongodConfigPath, minimalConfigPath, backupName, restoreMongodVersion,
	rsName, rsMembers string,
) error {
	config, err := binary.CreateMongodConfig(mongodConfigPath)
	if err != nil {
		return err
	}

	localStorage := binary.CreateLocalStorage(config.GetDBPath())

	uploader, err := internal.ConfigureUploader()
	if err != nil {
		return err
	}
	uploader.ChangeDirectory(utility.BaseBackupPath + "/")

	if minimalConfigPath == "" {
		minimalConfigPath, err = config.SaveConfigToTempFile("storage", "systemLog")
		if err != nil {
			return err
		}
	}

	restoreService, err := binary.CreateRestoreService(ctx, localStorage, uploader, minimalConfigPath)
	if err != nil {
		return err
	}

	rsConfig := binary.RsConfig{RsName: rsName, RsMembers: rsMembers}
	if err = rsConfig.Validate(); err != nil {
		return err
	}
	// check backup existence and resolve flag LATEST
	backup, err := internal.GetBackupByName(backupName, "", uploader.Folder())
	if err != nil {
		return err
	}

	return restoreService.DoRestore(backup.Name, restoreMongodVersion, rsConfig)
}
