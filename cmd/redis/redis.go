package redis

import (
	"fmt"
	"os"
	"strings"

	"github.com/apecloud/dataprotection-wal-g/cmd/common"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

var ShortDescription = "Redis backup tool"

// These variables are here only to show current version. They are set in makefile during build process
var walgVersion = "devel"
var gitRevision = "devel"
var buildDate = "devel"

var cmd = &cobra.Command{
	Use:     "redis",
	Short:   ShortDescription, // TODO : improve description
	Version: strings.Join([]string{walgVersion, gitRevision, buildDate, "Redis"}, "\t"),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := internal.AssertRequiredSettingsSet()
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	common.Init(cmd, internal.REDIS)
}
