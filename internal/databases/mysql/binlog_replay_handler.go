package mysql

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
)

const binlogFetchAhead = 2

type replayHandler struct {
	logCh chan string
	errCh chan error
	endTS string
}

func newReplayHandler(endTS time.Time) *replayHandler {
	rh := new(replayHandler)
	rh.endTS = endTS.Local().Format(TimeMysqlFormat)
	rh.logCh = make(chan string, binlogFetchAhead)
	rh.errCh = make(chan error, 1)
	go rh.replayLogs()
	return rh
}

func (rh *replayHandler) replayLogs() {
	for binlogPath := range rh.logCh {
		tracelog.InfoLogger.Printf("replaying %s ...", path.Base(binlogPath))
		err := rh.replayLog(binlogPath)
		os.Remove(binlogPath)
		if err != nil {
			tracelog.ErrorLogger.Printf("failed to replay %s: %v", path.Base(binlogPath), err)
			rh.errCh <- err
			break
		}
	}
	close(rh.errCh)
}

func (rh *replayHandler) replayLog(binlogPath string) error {
	cmd, err := internal.GetCommandSetting(internal.MysqlBinlogReplayCmd)
	if err != nil {
		return err
	}
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("%s=%s", "WALG_MYSQL_CURRENT_BINLOG", binlogPath),
		fmt.Sprintf("%s=%s", "WALG_MYSQL_BINLOG_END_TS", rh.endTS))
	cmd.Env = env
	return cmd.Run()
}

func (rh *replayHandler) wait() error {
	close(rh.logCh)
	return <-rh.errCh
}

func (rh *replayHandler) handleBinlog(binlogPath string) error {
	select {
	case err := <-rh.errCh:
		return err
	case rh.logCh <- binlogPath:
		return nil
	}
}

func HandleBinlogReplay(folder storage.Folder, backupName string, untilTS string, untilBinlogLastModifiedTS string, sinceTS string) {
	dstDir, err := internal.GetLogsDstSettings(internal.MysqlBinlogDstSetting)
	tracelog.ErrorLogger.FatalOnError(err)

	startTS, endTS, endBinlogTS, err := getTimestamps(folder, backupName, untilTS, untilBinlogLastModifiedTS, sinceTS)
	tracelog.ErrorLogger.FatalOnError(err)

	handler := newReplayHandler(endTS)

	tracelog.InfoLogger.Printf("Fetching binlogs since %s until %s", startTS, endTS)
	err = fetchLogs(folder, dstDir, startTS, endTS, endBinlogTS, handler)
	tracelog.ErrorLogger.FatalfOnError("Failed to fetch binlogs: %v", err)

	err = handler.wait()
	tracelog.ErrorLogger.FatalfOnError("Failed to apply binlogs: %v", err)
}

func getTimestamps(folder storage.Folder, backupName, untilTS, untilBinlogLastModifiedTS, sinceTS string) (time.Time, time.Time, time.Time, error) {
	var startTS time.Time
	if sinceTS != "" {
		var err error
		startTS, err = utility.ParseUntilTS(sinceTS)
		if err != nil {
			return time.Time{}, time.Time{}, time.Time{}, err
		}
	} else {
		// get the latest base backup
		backup, err := internal.GetBackupByName(backupName, utility.BaseBackupPath, folder)
		if err != nil {
			return time.Time{}, time.Time{}, time.Time{}, errors.Wrap(err, "Unable to get backup")
		}

		startTS, err = getBinlogSinceTS(folder, backup)
		if err != nil {
			return time.Time{}, time.Time{}, time.Time{}, err
		}
	}
	endTS, err := utility.ParseUntilTS(untilTS)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, err
	}

	endBinlogTS, err := utility.ParseUntilTS(untilBinlogLastModifiedTS)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, err
	}
	return startTS, endTS, endBinlogTS, nil
}
