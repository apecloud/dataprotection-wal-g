package ammogenerator

import (
	"fmt"
	"os"

	"github.com/apecloud/dataprotection-wal-g/tests_func/mongodb/mongoload/internal"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

var Cmd = &cobra.Command{
	Use:   "TODO", // TODO: fill use
	Short: "Ammo generator tool",
	Run: func(cmd *cobra.Command, args []string) {
		err := internal.GenerateAmmo(os.Stdin, os.Stdout)
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func Execute() {
	if err := Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
