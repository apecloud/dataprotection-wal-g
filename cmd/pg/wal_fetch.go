package pg

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"
	"github.com/apecloud/dataprotection-wal-g/internal/multistorage"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const WalFetchShortDescription = "Fetches a WAL file from storage"

// walFetchCmd represents the walFetch command
var walFetchCmd = &cobra.Command{
	Use:   "wal-fetch wal_name destination_filename",
	Short: WalFetchShortDescription, // TODO : improve description
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		folder, err := internal.ConfigureFolder()
		tracelog.ErrorLogger.FatalOnError(err)

		failover, err := internal.InitFailoverStorages()
		tracelog.ErrorLogger.FatalOnError(err)

		folderReader, err := multistorage.NewStorageFolderReader(folder, failover)
		tracelog.ErrorLogger.FatalOnError(err)

		postgres.HandleWALFetch(folderReader, args[0], args[1], true)
	},
}

func init() {
	Cmd.AddCommand(walFetchCmd)
}
