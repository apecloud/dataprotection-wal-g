package gp

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/greenplum"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const (
	segmentCmdRunShortDescription = "Runs the provided cmd on segment and updates the state file"
)

var (
	segCmdRunCmd = &cobra.Command{
		Use:   "seg-cmd-run name args --content-id=[content_id]",
		Short: segmentCmdRunShortDescription, // TODO : improve description
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			cmdName := args[0]
			cmdArgs := args[1]

			greenplum.SetSegmentStoragePrefix(contentID)

			stateUpdateInterval, err := internal.GetDurationSetting(internal.GPSegmentsUpdInterval)
			tracelog.ErrorLogger.FatalOnError(err)
			greenplum.NewSegCmdRunner(contentID, cmdName, cmdArgs, stateUpdateInterval).Run()
		},
	}
)

var contentID int

func init() {
	segCmdRunCmd.PersistentFlags().IntVar(&contentID, "content-id", 0, "segment content ID")
	_ = segCmdRunCmd.MarkFlagRequired("content-id")
	// Since this is a utility command, it should not be exposed to the end user.
	segCmdRunCmd.Hidden = true
	cmd.AddCommand(segCmdRunCmd)
}
