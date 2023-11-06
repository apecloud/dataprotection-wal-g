package mysql

import (
	"os"
	"os/exec"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/limiters"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

func HandleBackupPush(folder storage.Folder, uploader internal.Uploader,
	backupCmd *exec.Cmd, isPermanent bool, userDataRaw string) {
	db, err := getMySQLConnection()
	tracelog.ErrorLogger.FatalOnError(err)
	defer utility.LoggedClose(db, "")

	flavor, err := getMySQLFlavor(db)
	tracelog.ErrorLogger.FatalOnError(err)

	gtidStart, err := getMySQLGTIDExecuted(db, flavor)
	tracelog.ErrorLogger.FatalOnError(err)

	binlogStart, err := getLastUploadedBinlogBeforeGTID(folder, gtidStart, flavor)
	tracelog.ErrorLogger.FatalfOnError("failed to get last uploaded binlog: %v", err)
	timeStart := utility.TimeNowCrossPlatformLocal()

	stdout, stderr, err := utility.StartCommandWithStdoutStderr(backupCmd)
	tracelog.ErrorLogger.FatalfOnError("failed to start backup create command: %v", err)

	fileName, err := uploader.PushStream(limiters.NewDiskLimitReader(stdout))
	tracelog.ErrorLogger.FatalfOnError("failed to push backup: %v", err)

	err = backupCmd.Wait()
	if err != nil {
		tracelog.ErrorLogger.Printf("Backup command output:\n%s", stderr.String())
		tracelog.ErrorLogger.Fatalf("backup create command failed: %v", err)
	}

	binlogEnd, err := getLastUploadedBinlog(folder)
	tracelog.ErrorLogger.FatalfOnError("failed to get last uploaded binlog (after): %v", err)
	timeStop := utility.TimeNowCrossPlatformLocal()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
		tracelog.WarningLogger.Printf("Failed to obtain the OS hostname for the backup sentinel\n")
	}

	uploadedSize, err := uploader.UploadedDataSize()
	if err != nil {
		tracelog.ErrorLogger.Printf("Failed to calc uploaded data size: %v", err)
	}

	rawSize, err := uploader.RawDataSize()
	if err != nil {
		tracelog.ErrorLogger.Printf("Failed to calc raw data size: %v", err)
	}

	userData, err := internal.UnmarshalSentinelUserData(userDataRaw)
	tracelog.ErrorLogger.FatalfOnError("Failed to unmarshal the provided UserData: %s", err)

	sentinel := StreamSentinelDto{
		BinLogStart:      binlogStart,
		BinLogEnd:        binlogEnd,
		StartLocalTime:   timeStart,
		StopLocalTime:    timeStop,
		Hostname:         hostname,
		CompressedSize:   uploadedSize,
		UncompressedSize: rawSize,
		IsPermanent:      isPermanent,
		UserData:         userData,
	}
	tracelog.InfoLogger.Printf("Backup sentinel: %s", sentinel.String())

	err = internal.UploadSentinel(uploader, &sentinel, fileName)
	tracelog.ErrorLogger.FatalOnError(err)
}
