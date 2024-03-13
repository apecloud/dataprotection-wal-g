package mysql

import (
	"time"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mysql"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const (
	binlogServerShortDescription = "Create server for backup slaves"
	binlogSinceFlagShortDescr    = "backup name starting from which you want to use binlogs"
	untilFlagShortDescr          = "time in RFC3339 for PITR"
	sinceTSFlagShortDescr        = "binlog starting timestamp from which you want to use binlogs"
)

var sinceTS string
var untilTS string
var BinlogBackupName string

var (
	binlogServerCmd = &cobra.Command{
		Use:   "binlog-server",
		Short: binlogServerShortDescription,
		Args:  cobra.NoArgs,
		PreRun: func(cmd *cobra.Command, args []string) {
			internal.RequiredSettings[internal.MysqlBinlogServerHost] = true
			internal.RequiredSettings[internal.MysqlBinlogServerPort] = true
			internal.RequiredSettings[internal.MysqlBinlogServerUser] = true
			internal.RequiredSettings[internal.MysqlBinlogServerPassword] = true
			internal.RequiredSettings[internal.MysqlBinlogServerID] = true
			internal.RequiredSettings[internal.MysqlBinlogServerReplicaSource] = true
			err := internal.AssertRequiredSettingsSet()
			tracelog.ErrorLogger.FatalOnError(err)
		},
		Run: func(cmd *cobra.Command, args []string) {
			mysql.HandleBinlogServer(BinlogBackupName, untilTS, sinceTS)
		},
	}
)

func init() {
	binlogServerCmd.Flags().StringVar(&sinceTS,
		"since-ts",
		"",
		sinceTSFlagShortDescr)
	binlogServerCmd.Flags().StringVar(&BinlogBackupName, "since", "LATEST", binlogSinceFlagShortDescr)
	binlogServerCmd.Flags().StringVar(&untilTS,
		"until",
		utility.TimeNowCrossPlatformUTC().Format(time.RFC3339),
		untilFlagShortDescr)
	cmd.AddCommand(binlogServerCmd)
}
