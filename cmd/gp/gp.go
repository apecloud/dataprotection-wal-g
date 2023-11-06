package gp

import (
	"fmt"
	"os"
	"strings"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/greenplum"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"
	"github.com/spf13/viper"

	"github.com/apecloud/dataprotection-wal-g/cmd/common"

	"github.com/apecloud/dataprotection-wal-g/cmd/pg"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

var dbShortDescription = "GreenplumDB backup tool"

// These variables are here only to show current version. They are set in makefile during build process
var walgVersion = "devel"
var gitRevision = "devel"
var buildDate = "devel"

var cmd = &cobra.Command{
	Use:     "wal-g",
	Short:   dbShortDescription, // TODO : improve description
	Version: strings.Join([]string{walgVersion, gitRevision, buildDate, "GreenplumDB"}, "\t"),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Greenplum uses the 64MB WAL segment size by default
		postgres.SetWalSize(viper.GetUint64(internal.PgWalSize))
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

var SegContentID string

func init() {
	common.Init(cmd, internal.GP)

	_ = cmd.MarkFlagRequired("config") // config is required for Greenplum WAL-G
	// wrap the Postgres command so it can be used in the same binary
	wrappedPgCmd := pg.Cmd
	wrappedPgCmd.Use = "seg"
	wrappedPgCmd.Short = "PostgreSQL command series to run on segments (use with caution)"
	wrappedPreRun := wrappedPgCmd.PersistentPreRun
	wrappedPgCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// segment content ID is required in order to get the corresponding segment subfolder
		contentID, err := greenplum.ConfigureSegContentID(SegContentID)
		tracelog.ErrorLogger.FatalOnError(err)
		greenplum.SetSegmentStoragePrefix(contentID)
		wrappedPreRun(cmd, args)
	}
	wrappedPgCmd.PersistentFlags().StringVar(&SegContentID, "content-id", "", "segment content ID")
	cmd.AddCommand(wrappedPgCmd)

	// Add the hidden prefetch command to the root command
	// since WAL-G prefetch fork logic does not know anything about the "wal-g seg" subcommand
	pg.WalPrefetchCmd.PreRun = func(cmd *cobra.Command, args []string) {
		internal.RequiredSettings[internal.StoragePrefixSetting] = true
		tracelog.ErrorLogger.FatalOnError(internal.AssertRequiredSettingsSet())
	}
	cmd.AddCommand(pg.WalPrefetchCmd)
}
