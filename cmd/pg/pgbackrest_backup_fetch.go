package pg

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres/pgbackrest"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

var pgbackrestBackupFetchCmd = &cobra.Command{
	Use:   "backup-fetch destination-directory backup-name",
	Short: backupFetchShortDescription,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		internal.ConfigureLimiters()

		destinationDirectory := args[0]
		backupName := args[1]
		folder, stanza := configurePgbackrestSettings()
		backupSelector := pgbackrest.NewBackupSelector(backupName, stanza)
		err := pgbackrest.HandlePgbackrestBackupFetch(folder, stanza, destinationDirectory, backupSelector)
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func init() {
	pgbackrestCmd.AddCommand(pgbackrestBackupFetchCmd)
}
