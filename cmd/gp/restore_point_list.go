package gp

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/greenplum"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const (
	restorePointListShortDescription = "Prints available restore points"
)

var (
	// restorePointListCmd represents the restorePointList command
	restorePointListCmd = &cobra.Command{
		Use:   "restore-point-list",
		Short: restorePointListShortDescription, // TODO : improve description
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			folder, err := internal.ConfigureFolder()
			tracelog.ErrorLogger.FatalOnError(err)
			greenplum.HandleRestorePointList(folder.GetSubFolder(utility.BaseBackupPath), pretty, jsonOutput)
		},
	}
)

func init() {
	cmd.AddCommand(restorePointListCmd)

	restorePointListCmd.Flags().BoolVar(&pretty, PrettyFlag, false, "Prints more readable output")
	restorePointListCmd.Flags().BoolVar(&jsonOutput, JSONFlag, false, "Prints output in json format")
}
