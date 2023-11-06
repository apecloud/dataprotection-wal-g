package mongo

import (
	"io"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/common"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
)

// HandleBackupShow prints sentinel contents.
func HandleBackupShow(backupFolder storage.Folder, backupName string, output io.Writer, pretty bool) (err error) {
	sentinel, err := common.DownloadSentinel(backupFolder, backupName)
	if err != nil {
		return err
	}

	return internal.WriteAsJSON(sentinel, output, pretty)
}
