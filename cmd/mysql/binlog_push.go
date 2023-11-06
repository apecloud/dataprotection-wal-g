package mysql

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mysql"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const binlogPushShortDescription = "Upload binlogs to the storage"

var untilBinlog string

// binlogPushCmd represents the cron command
var binlogPushCmd = &cobra.Command{
	Use:   "binlog-push",
	Short: binlogPushShortDescription,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		uploader, err := internal.ConfigureUploader()
		tracelog.ErrorLogger.FatalOnError(err)
		checkGTIDs, _ := internal.GetBoolSettingDefault(internal.MysqlCheckGTIDs, false)
		mysql.HandleBinlogPush(uploader, untilBinlog, checkGTIDs)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		internal.RequiredSettings[internal.MysqlDatasourceNameSetting] = true
		err := internal.AssertRequiredSettingsSet()
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func init() {
	cmd.AddCommand(binlogPushCmd)
	binlogPushCmd.Flags().StringVar(&untilBinlog, "until", "", "binlog file name to stop at. Current active by default")
}
