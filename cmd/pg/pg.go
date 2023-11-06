package pg

import (
	"fmt"
	"os"
	"strings"

	"github.com/apecloud/dataprotection-wal-g/cmd/common"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wal-g/tracelog"
)

const WalgShortDescription = "PostgreSQL backup tool"

var (
	// These variables are here only to show current version. They are set in makefile during build process
	walgVersion = "devel"
	gitRevision = "devel"
	buildDate   = "devel"

	Cmd = &cobra.Command{
		Use:     "wal-g",
		Short:   WalgShortDescription, // TODO : improve short and long descriptions
		Version: strings.Join([]string{walgVersion, gitRevision, buildDate, "PostgreSQL"}, "\t"),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := internal.AssertRequiredSettingsSet()
			tracelog.ErrorLogger.FatalOnError(err)

			if viper.IsSet(internal.PgWalSize) {
				postgres.SetWalSize(viper.GetUint64(internal.PgWalSize))
			}
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the PgCmd.
func Execute() {
	configureCommand()
	if err := Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func configureCommand() {
	common.Init(Cmd, internal.PG)
	internal.AddTurboFlag(Cmd)
}
