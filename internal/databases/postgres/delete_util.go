package postgres

import (
	"strings"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

func GetPermanentBackupsAndWals(folder storage.Folder) (map[string]bool, map[string]bool) {
	tracelog.InfoLogger.Println("retrieving permanent objects")
	backupTimes, err := internal.GetBackups(folder.GetSubFolder(utility.BaseBackupPath))
	if err != nil {
		return map[string]bool{}, map[string]bool{}
	}

	permanentBackups := map[string]bool{}
	permanentWals := map[string]bool{}
	for _, backupTime := range backupTimes {
		backup := NewBackup(folder.GetSubFolder(utility.BaseBackupPath), backupTime.BackupName)
		meta, err := backup.FetchMeta()
		if err != nil {
			internal.FatalOnUnrecoverableMetadataError(backupTime, err)
			continue
		}
		if meta.IsPermanent {
			timelineID, err := ParseTimelineFromBackupName(backup.Name)
			if err != nil {
				tracelog.ErrorLogger.Printf("failed to parse backup timeline for backup %s with error %s, ignoring...",
					backupTime.BackupName, err.Error())
				continue
			}

			startWalSegmentNo := newWalSegmentNo(meta.StartLsn - 1)
			endWalSegmentNo := newWalSegmentNo(meta.FinishLsn - 1)
			for walSegmentNo := startWalSegmentNo; walSegmentNo <= endWalSegmentNo; walSegmentNo = walSegmentNo.next() {
				permanentWals[walSegmentNo.getFilename(timelineID)] = true
			}
			permanentBackups[backupTime.BackupName] = true
		}
	}
	if len(permanentBackups) > 0 {
		tracelog.InfoLogger.Printf("Found permanent objects: backups=%v, wals=%v\n",
			permanentBackups, permanentWals)
	}
	return permanentBackups, permanentWals
}

func IsPermanent(objectName string, permanentBackups, permanentWals map[string]bool) bool {
	if strings.HasPrefix(objectName, utility.WalPath) && len(objectName) >= len(utility.WalPath)+24 {
		wal := objectName[len(utility.WalPath) : len(utility.WalPath)+24]
		return permanentWals[wal]
	}
	if strings.HasPrefix(objectName, utility.BaseBackupPath) {
		backup := utility.StripLeftmostBackupName(objectName[len(utility.BaseBackupPath):])
		return permanentBackups[backup]
	}
	// should not reach here, default to false
	return false
}
