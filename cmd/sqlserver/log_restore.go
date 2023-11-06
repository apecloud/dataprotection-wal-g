package sqlserver

import (
	"time"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/sqlserver"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/cobra"
)

const logRestoreShortDescription = "Restores log from the storage"

var logRestoreBackupName string
var logRestoreUntilTS string
var logRestoreDatabases []string
var logRestoreFrom []string
var logRestoreNoRecovery bool

var logRestoreCmd = &cobra.Command{
	Use:   "log-restore log-name",
	Short: logRestoreShortDescription,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		sqlserver.HandleLogRestore(logRestoreBackupName,
			logRestoreUntilTS, logRestoreDatabases, logRestoreFrom, logRestoreNoRecovery)
	},
}

func init() {
	logRestoreCmd.PersistentFlags().StringVar(&logRestoreBackupName, "since", "LATEST",
		"backup name starting from which you want to restore logs")
	logRestoreCmd.PersistentFlags().StringVar(&logRestoreUntilTS, "until",
		utility.TimeNowCrossPlatformUTC().Format(time.RFC3339), "time in RFC3339 for PITR")
	logRestoreCmd.PersistentFlags().StringSliceVarP(&logRestoreDatabases, "databases", "d", []string{},
		"List of databases to restore logs. All non-system databases from backup as default")
	logRestoreCmd.PersistentFlags().StringSliceVarP(&logRestoreFrom, "from", "f", []string{},
		"List of source database to restore logs from. By default it's the same as list of database, "+
			"those every database log is restored from self backup")
	logRestoreCmd.PersistentFlags().BoolVarP(&logRestoreNoRecovery, "no-recovery", "n", false,
		"Restore with NO_RECOVERY option")
	cmd.AddCommand(logRestoreCmd)
}
