package pg

import (
	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres/pgbackrest"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

var pgbackrestWalFetchCmd = &cobra.Command{
	Use:   "wal-fetch wal_name destination_filename",
	Short: WalFetchShortDescription,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		folder, stanza := configurePgbackrestSettings()
		err := pgbackrest.HandleWalFetch(folder, stanza, args[0], args[1])
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func init() {
	pgbackrestCmd.AddCommand(pgbackrestWalFetchCmd)
}
