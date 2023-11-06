package mongo

import (
	"context"
	"os"
	"syscall"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/archive"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/client"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const (
	backupPushShortDescription = "Pushes backup to storage"
	PermanentFlag              = "permanent"
	PermanentShorthand         = "p"
)

var (
	permanent = false
)

// backupPushCmd represents the backupPush command
var backupPushCmd = &cobra.Command{
	Use:   "backup-push",
	Short: backupPushShortDescription,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		internal.ConfigureLimiters()

		ctx, cancel := context.WithCancel(context.Background())
		signalHandler := utility.NewSignalHandler(ctx, cancel, []os.Signal{syscall.SIGINT, syscall.SIGTERM})
		defer func() { _ = signalHandler.Close() }()

		mongodbURL, err := internal.GetRequiredSetting(internal.MongoDBUriSetting)
		tracelog.ErrorLogger.FatalOnError(err)

		// set up mongodb client and oplog fetcher
		mongoClient, err := client.NewMongoClient(ctx, mongodbURL)
		tracelog.ErrorLogger.FatalOnError(err)

		uplProvider, err := internal.ConfigureSplitUploader()
		tracelog.ErrorLogger.FatalOnError(err)
		uplProvider.ChangeDirectory(utility.BaseBackupPath)

		backupCmd, err := internal.GetCommandSettingContext(ctx, internal.NameStreamCreateCmd)
		tracelog.ErrorLogger.FatalOnError(err)
		backupCmd.Stderr = os.Stderr
		uploader := archive.NewStorageUploader(uplProvider)
		metaConstructor := archive.NewBackupMongoMetaConstructor(ctx, mongoClient, uplProvider.Folder(), permanent)

		err = mongo.HandleBackupPush(uploader, metaConstructor, backupCmd)
		tracelog.ErrorLogger.FatalfOnError("Backup creation failed: %v", err)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		internal.RequiredSettings[internal.NameStreamCreateCmd] = true
		err := internal.AssertRequiredSettingsSet()
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func init() {
	backupPushCmd.Flags().BoolVarP(&permanent, PermanentFlag, PermanentShorthand, false, "Pushes permanent backup")
	cmd.AddCommand(backupPushCmd)
}
