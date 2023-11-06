package mongo

import (
	"context"
	"os"
	"syscall"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/archive"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const backupDeleteShortDescription = "Deletes backup data from storage"

var (
	confirmedBackupDelete bool
)

// backupDeleteCmd represents the backupDelete command
var backupDeleteCmd = &cobra.Command{
	Use:   "backup-delete <backup-name>",
	Short: backupDeleteShortDescription,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		signalHandler := utility.NewSignalHandler(ctx, cancel, []os.Signal{syscall.SIGINT, syscall.SIGTERM})
		defer func() { _ = signalHandler.Close() }()

		backupName := args[0]

		// set up storage downloader client
		downloader, err := archive.NewStorageDownloader(archive.NewDefaultStorageSettings())
		tracelog.ErrorLogger.FatalOnError(err)

		// set up storage downloader client
		purger, err := archive.NewStoragePurger(archive.NewDefaultStorageSettings())
		tracelog.ErrorLogger.FatalOnError(err)

		err = mongo.HandleBackupDelete(backupName, downloader, purger, !confirmedBackupDelete)
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func init() {
	backupDeleteCmd.Flags().BoolVar(&confirmedBackupDelete, internal.ConfirmFlag, false, "Confirms backup deletion")
	cmd.AddCommand(backupDeleteCmd)
}
