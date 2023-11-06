package pg

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const pgbackrestCommandDescription = "Interact with pgbackrest backups (beta)"

var pgbackrestCmd = &cobra.Command{
	Use:   "pgbackrest",
	Short: pgbackrestCommandDescription,
}

func init() {
	Cmd.AddCommand(pgbackrestCmd)
}

func configurePgbackrestSettings() (folder storage.Folder, stanza string) {
	folder, err := internal.ConfigureFolder()
	tracelog.ErrorLogger.FatalOnError(err)
	stanza, _ = internal.GetSetting(internal.PgBackRestStanza)
	return folder, stanza
}
