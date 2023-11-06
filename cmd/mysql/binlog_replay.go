package mysql

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mysql"
	"github.com/apecloud/dataprotection-wal-g/utility"
)

const replaySinceFlagShortDescr = "backup name starting from which you want to fetch binlogs"
const replayUntilFlagShortDescr = "time in RFC3339 for PITR"
const replayUntilBinlogLastModifiedFlagShortDescr = "time in RFC3339 that is used to prevent wal-g from replaying" +
	" binlogs that was created/modified after this time"

var replayBackupName string
var replayUntilTS string
var replayUntilBinlogLastModifiedTS string
var replaySinceTS string

var binlogReplayCmd = &cobra.Command{
	Use:   "binlog-replay",
	Short: "Fetch binlogs from storage and replays them to MySQL",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		folder, err := internal.ConfigureFolder()
		tracelog.ErrorLogger.FatalOnError(err)
		mysql.HandleBinlogReplay(folder, replayBackupName, replayUntilTS, replayUntilBinlogLastModifiedTS, replaySinceTS)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		internal.RequiredSettings[internal.MysqlBinlogReplayCmd] = true
		err := internal.AssertRequiredSettingsSet()
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func init() {
	binlogReplayCmd.PersistentFlags().StringVar(&replayBackupName, "since", "LATEST", replaySinceFlagShortDescr)
	binlogReplayCmd.PersistentFlags().StringVar(&replayUntilTS, "until",
		utility.TimeNowCrossPlatformUTC().Format(time.RFC3339), replayUntilFlagShortDescr)
	binlogReplayCmd.PersistentFlags().StringVar(&replayUntilBinlogLastModifiedTS, "until-binlog-last-modified-time",
		"", replayUntilBinlogLastModifiedFlagShortDescr)
	binlogReplayCmd.PersistentFlags().StringVar(&replaySinceTS, "since-time",
		"", "binlog since time in RFC3339")
	cmd.AddCommand(binlogReplayCmd)
}
