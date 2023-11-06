package redis

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/redis"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/redis/archive"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

var (
	permanent = false
)

const (
	backupPushShortDescription = "Makes backup and uploads it to storage"
	PermanentFlag              = "permanent"
	PermanentShorthand         = "p"
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

		uploader, err := internal.ConfigureUploader()
		tracelog.ErrorLogger.FatalOnError(err)

		// Configure folder
		uploader.ChangeDirectory(utility.BaseBackupPath)

		backupCmd, err := internal.GetCommandSettingContext(ctx, internal.NameStreamCreateCmd)
		tracelog.ErrorLogger.FatalOnError(err)

		redisPassword, ok := internal.GetSetting(internal.RedisPassword)
		if ok && redisPassword != "" { // special hack for redis-cli
			backupCmd.Env = append(backupCmd.Env, fmt.Sprintf("REDISCLI_AUTH=%s", redisPassword))
		}
		backupCmd.Stderr = os.Stderr
		metaConstructor := archive.NewBackupRedisMetaConstructor(ctx, uploader.Folder(), permanent)

		err = redis.HandleBackupPush(uploader, backupCmd, metaConstructor)
		tracelog.ErrorLogger.FatalfOnError("Redis backup creation failed: %v", err)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		internal.RequiredSettings[internal.NameStreamCreateCmd] = true
		err := internal.AssertRequiredSettingsSet()
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func init() {
	backupPushCmd.Flags().BoolVarP(&permanent, PermanentFlag, PermanentShorthand, false, "Pushes backup with 'permanent' flag")
	cmd.AddCommand(backupPushCmd)
}
