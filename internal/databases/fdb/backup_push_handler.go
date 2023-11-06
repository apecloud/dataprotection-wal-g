package fdb

import (
	"os/exec"
	"time"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

// TODO: add more metadata
type streamSentinelDto struct {
	StartLocalTime time.Time
}

func HandleBackupPush(uploader internal.Uploader, backupCmd *exec.Cmd) {
	timeStart := utility.TimeNowCrossPlatformLocal()

	stdout, stderr, err := utility.StartCommandWithStdoutStderr(backupCmd)
	tracelog.ErrorLogger.FatalfOnError("failed to start backup create command: %v", err)

	fileName, err := uploader.PushStream(stdout)
	tracelog.ErrorLogger.FatalfOnError("failed to push backup: %v", err)

	err = backupCmd.Wait()
	if err != nil {
		tracelog.ErrorLogger.Printf("Backup command output:\n%s", stderr.String())
		tracelog.ErrorLogger.Fatalf("backup create command failed: %v", err)
	}

	sentinel := streamSentinelDto{StartLocalTime: timeStart}

	err = internal.UploadSentinel(uploader, &sentinel, fileName)
	tracelog.ErrorLogger.FatalOnError(err)
}
