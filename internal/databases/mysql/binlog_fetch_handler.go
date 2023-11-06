package mysql

import (
	"os"
	"path"
	"path/filepath"

	"github.com/wal-g/tracelog"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
)

type indexHandler struct {
	dstDir  string
	binlogs []string
}

func newIndexHandler(dstDir string) *indexHandler {
	ih := new(indexHandler)
	ih.dstDir = dstDir
	return ih
}

func (ih *indexHandler) handleBinlog(binlogPath string) error {
	ih.binlogs = append(ih.binlogs, path.Base(binlogPath))
	return nil
}

func (ih *indexHandler) createIndexFile() error {
	indexFile, err := os.Create(filepath.Join(ih.dstDir, "binlogs_order"))
	if err != nil {
		return err
	}
	defer indexFile.Close()
	for _, binlog := range ih.binlogs {
		_, err = indexFile.WriteString(binlog + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func HandleBinlogFetch(folder storage.Folder, backupName string, untilTS string, untilBinlogLastModifiedTS string, sinceTS string) {
	dstDir, err := internal.GetLogsDstSettings(internal.MysqlBinlogDstSetting)
	tracelog.ErrorLogger.FatalOnError(err)

	startTS, endTS, endBinlogTS, err := getTimestamps(folder, backupName, untilTS, untilBinlogLastModifiedTS, sinceTS)
	tracelog.ErrorLogger.FatalOnError(err)

	handler := newIndexHandler(dstDir)

	tracelog.InfoLogger.Printf("Fetching binlogs since %s until %s", startTS, endTS)
	err = fetchLogs(folder, dstDir, startTS, endTS, endBinlogTS, handler)
	tracelog.ErrorLogger.FatalfOnError("Failed to fetch binlogs: %v", err)

	err = handler.createIndexFile()
	tracelog.ErrorLogger.FatalfOnError("Failed to create binlog index file: %v", err)
}
