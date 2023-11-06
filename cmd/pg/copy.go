package pg

import (
	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"
	"github.com/spf13/cobra"
)

const (
	backupCopyUsage            = "copy"
	backupCopyShortDescription = "copy specific or all backups"
	backupCopyLongDescription  = "Copy backup(s) from one storage to another according to configs " +
		"(with history by default)"

	backupNameFlag        = "backup-name"
	backupNameShorthand   = "b"
	backupNameDescription = "Copy specific backup"

	fromFlag        = "from"
	fromShorthand   = "f"
	fromDescription = "Storage config from where should copy backup"

	toFlag        = "to"
	toShorthand   = "t"
	toDescription = "Storage config to where should copy backup"

	withoutHistoryFlag        = "without-history"
	withoutHistoryShorthand   = "w"
	withoutHistoryDescription = "Copy backup without history"
)

var (
	backupName     string
	fromConfigFile string
	toConfigFile   string
	withoutHistory = false

	backupCopyCmd = &cobra.Command{
		Use:   backupCopyUsage,
		Short: backupCopyShortDescription,
		Long:  backupCopyLongDescription,
		Args:  cobra.ExactArgs(0),
		Run:   runBackupCopy,
		PersistentPreRun: func(*cobra.Command, []string) {
			// do not check for any configured settings because wal-g copy uses the different
			// settings init process
		},
	}
)

func runBackupCopy(cmd *cobra.Command, args []string) {
	postgres.HandleCopy(fromConfigFile, toConfigFile, backupName, withoutHistory)
}

func init() {
	Cmd.AddCommand(backupCopyCmd)

	backupCopyCmd.Flags().StringVarP(&backupName, backupNameFlag, backupNameShorthand, "", backupNameDescription)
	backupCopyCmd.Flags().StringVarP(&toConfigFile, toFlag, toShorthand, "", toDescription)
	backupCopyCmd.Flags().StringVarP(&fromConfigFile, fromFlag, fromShorthand, "", fromDescription)
	backupCopyCmd.Flags().BoolVarP(&withoutHistory,
		withoutHistoryFlag,
		withoutHistoryShorthand,
		false,
		withoutHistoryDescription)

	_ = backupCopyCmd.MarkFlagRequired(toFlag)
	_ = backupCopyCmd.MarkFlagRequired(fromFlag)
}
