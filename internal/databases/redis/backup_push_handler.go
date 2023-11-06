package redis

import (
	"os/exec"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/redis/archive"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

func HandleBackupPush(uploader internal.Uploader, backupCmd *exec.Cmd, metaConstructor internal.MetaConstructor) error {
	stdout, err := utility.StartCommandWithStdoutPipe(backupCmd)
	tracelog.ErrorLogger.FatalfOnError("failed to start backup create command: %v", err)

	redisUploader := archive.NewRedisStorageUploader(uploader)

	return redisUploader.UploadBackup(stdout, backupCmd, metaConstructor)
}
