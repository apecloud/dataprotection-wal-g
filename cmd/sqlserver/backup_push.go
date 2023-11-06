package sqlserver

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/sqlserver"
	"github.com/spf13/cobra"
)

const backupPushShortDescription = "Creates new backup and pushes it to the storage"

var backupPushDatabases []string
var backupUpdateLatest bool

var backupPushCmd = &cobra.Command{
	Use:   "backup-push",
	Short: backupPushShortDescription,
	Run: func(cmd *cobra.Command, args []string) {
		internal.ConfigureLimiters()
		sqlserver.HandleBackupPush(backupPushDatabases, backupUpdateLatest)
	},
}

func init() {
	backupPushCmd.PersistentFlags().StringSliceVarP(&backupPushDatabases, "databases", "d", []string{},
		"List of databases to backup. All not-system databases as default")
	backupPushCmd.PersistentFlags().BoolVarP(&backupUpdateLatest, "update-latest", "u", false,
		"Update latest backup instead of creating new one")
	cmd.AddCommand(backupPushCmd)
}
