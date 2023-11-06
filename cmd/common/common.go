package common

import (
	"strings"

	"github.com/apecloud/dataprotection-wal-g/cmd/common/st"
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/wal-g/tracelog"
)

const usageTemplate = `Usage:{{if .Runnable}}
{{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
{{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
{{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}` +
	// additional custom message : cli flags introduced by 'internal.AddConfigFlags()' are hidden by default
	`

To get the complete list of all global flags, run: 'wal-g flags'` +
	`{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

const hiddenConfigFlagAnnotation = "walg_annotation_hidden_config_flag"

func Init(cmd *cobra.Command, dbName string) {
	internal.ConfigureSettings(dbName)
	cobra.OnInitialize(internal.InitConfig, internal.Configure)

	cmd.InitDefaultVersionFlag()
	internal.AddConfigFlags(cmd, hiddenConfigFlagAnnotation)

	cmd.PersistentFlags().StringVar(&internal.CfgFile, "config", "", "config file (default is $HOME/.walg.json)")

	initHelp(cmd)

	// Add flags subcommand
	cmd.AddCommand(FlagsCmd)

	// Add completion subcommand
	cmd.AddCommand(CompletionCmd)

	// Add storage tools
	cmd.AddCommand(st.StorageToolsCmd)

	// profiler
	persistentPreRun := cmd.PersistentPreRun
	persistentPostRun := cmd.PersistentPostRun

	var p internal.ProfileStopper
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if persistentPreRun != nil {
			persistentPreRun(cmd, args)
		}

		var err error
		p, err = internal.Profile()
		tracelog.ErrorLogger.FatalOnError(err)
	}
	cmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		if persistentPostRun != nil {
			persistentPostRun(cmd, args)
		}

		// metrics hook
		internal.PushMetrics()

		if p != nil {
			p.Stop()
		}
	}

	// Don't run PersistentPreRun when shell autocompleting
	preRun := cmd.PersistentPreRun
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if strings.Index(cmd.Use, cobra.ShellCompRequestCmd) == 0 {
			return
		}
		preRun(cmd, args)
	}
}

// setup init and usage functionality
func initHelp(cmd *cobra.Command) {
	cmd.SetUsageTemplate(usageTemplate)
	defaultUsageFn := (&cobra.Command{}).UsageFunc()
	defaultHelpFn := (&cobra.Command{}).HelpFunc()

	// hide global config flags from usage output
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		hideGlobalConfigFlags(cmd)
		return defaultUsageFn(cmd)
	})

	// hide global config flags from help output
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		hideGlobalConfigFlags(cmd)
		defaultHelpFn(cmd, args)
	})

	// Init help subcommand
	cmd.InitDefaultHelpCmd()
	helpCmd, _, _ := cmd.Find([]string{"help"})
	// fix to disable the required settings check for the help subcommand
	helpCmd.PersistentPreRun = func(*cobra.Command, []string) {}
}

// hide global config flags from all subcommands except the "flags" subcommand
func hideGlobalConfigFlags(cmd *cobra.Command) {
	if cmd != FlagsCmd {
		cmd.Root().PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if _, ok := f.Annotations[hiddenConfigFlagAnnotation]; ok {
				f.Hidden = true
			}
		})
	}
}
