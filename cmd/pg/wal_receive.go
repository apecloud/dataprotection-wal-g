package pg

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/asm"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const walReceiveShortDescription = "Receive WAL stream with postgres Streaming Replication Protocol and push to storage"

// walReceiveCmd represents the walReceive command
var walReceiveCmd = &cobra.Command{
	Use:   "wal-receive",
	Short: walReceiveShortDescription,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		baseUploader, err := internal.ConfigureUploader()
		tracelog.ErrorLogger.FatalOnError(err)

		uploader, err := postgres.ConfigureWalUploader(baseUploader)
		tracelog.ErrorLogger.FatalOnError(err)

		archiveStatusManager, err := internal.ConfigureArchiveStatusManager()
		if err == nil {
			uploader.ArchiveStatusManager = asm.NewDataFolderASM(archiveStatusManager)
		} else {
			tracelog.ErrorLogger.PrintError(err)
			uploader.ArchiveStatusManager = asm.NewNopASM()
		}
		postgres.HandleWALReceive(uploader)
	},
}

func init() {
	Cmd.AddCommand(walReceiveCmd)
}
