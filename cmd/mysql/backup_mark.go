package mysql

import (
	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mysql"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const (
	BackupMarkShortDescription = "Marks a backup permanent or impermanent"
	BackupMarkLongDescription  = `Marks a backup permanent by default, or impermanent when flag is provided.
	Permanent backups are prevented from being removed when running delete.`
	ImpermanentDescription     = "Marks a backup impermanent"
	ImpermanentFlagShortHand   = "i"
	ImpermanentFlag            = "impermanent"
	backupMarkShortDescription = "mark permanent/impermanent target backup"
)

var (
	// backupMarkCmd represents the backupMark command
	backupMarkCmd = &cobra.Command{
		Use:   "backup-mark",
		Short: BackupMarkShortDescription,
		Long:  BackupMarkLongDescription,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			uploader, err := internal.ConfigureUploader()
			tracelog.ErrorLogger.FatalOnError(err)
			mysql.MarkBackup(uploader, name, !toImpermanent)
		},
	}
	toImpermanent = false
	name          = ""
)

func init() {
	backupMarkCmd.Flags().BoolVarP(&toImpermanent,
		ImpermanentFlag,
		ImpermanentFlagShortHand,
		false,
		ImpermanentDescription)
	backupMarkCmd.Flags().StringVarP(&name, backupNameFlag, backupShorthand, "", backupMarkShortDescription)
	cmd.AddCommand(backupMarkCmd)
}
