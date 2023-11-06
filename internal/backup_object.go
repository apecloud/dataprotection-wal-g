package internal

import (
	"time"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
)

func NewDefaultBackupObject(object storage.Object) BackupObject {
	return DefaultBackupObject{object}
}

type DefaultBackupObject struct {
	storage.Object
}

func (o DefaultBackupObject) GetBackupName() string {
	return utility.StripRightmostBackupName(o.GetName())
}

func (o DefaultBackupObject) GetBaseBackupName() string {
	return o.GetBackupName()
}

func (o DefaultBackupObject) GetIncrementFromName() string {
	return o.GetBackupName()
}

func (o DefaultBackupObject) IsFullBackup() bool {
	return true
}

func (o DefaultBackupObject) GetBackupTime() time.Time {
	return o.Object.GetLastModified()
}
