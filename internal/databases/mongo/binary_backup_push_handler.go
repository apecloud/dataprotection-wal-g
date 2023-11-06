package mongo

import (
	"context"
	"time"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/binary"
	"github.com/apecloud/dataprotection-wal-g/utility"
)

func HandleBinaryBackupPush(ctx context.Context, permanent bool, appName string) error {
	backupName := binary.GenerateNewBackupName()

	mongodbURI, err := internal.GetRequiredSetting(internal.MongoDBUriSetting)
	if err != nil {
		return err
	}
	mongodService, err := binary.CreateMongodService(ctx, appName, mongodbURI, 10*time.Minute)
	if err != nil {
		return err
	}

	uploader, err := internal.ConfigureUploader()
	if err != nil {
		return err
	}
	uploader.ChangeDirectory(utility.BaseBackupPath + "/")

	backupService, err := binary.CreateBackupService(ctx, mongodService, uploader)
	if err != nil {
		return err
	}

	return backupService.DoBackup(backupName, permanent)
}
