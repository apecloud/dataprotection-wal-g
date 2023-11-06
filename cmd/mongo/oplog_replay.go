package mongo

import (
	"context"
	"encoding/json"
	"os"
	"syscall"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/archive"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/client"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/models"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/oplog"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/stages"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
)

const LatestBackupString = "LATEST_BACKUP"

// oplogReplayCmd represents oplog replay procedure
var oplogReplayCmd = &cobra.Command{
	Use:   "oplog-replay <since ts.inc> <until ts.inc>",
	Short: "Fetches oplog archives from storage and applies to database",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		defer func() { tracelog.ErrorLogger.FatalOnError(err) }()

		ctx, cancel := context.WithCancel(context.Background())
		signalHandler := utility.NewSignalHandler(ctx, cancel, []os.Signal{syscall.SIGINT, syscall.SIGTERM})
		defer func() { _ = signalHandler.Close() }()

		replayArgs, err := buildOplogReplayRunArgs(args)
		if err != nil {
			return
		}

		err = runOplogReplay(ctx, replayArgs)
	},
}

type oplogReplayRunArgs struct {
	since models.Timestamp
	until models.Timestamp

	ignoreErrCodes map[string][]int32
	mongodbURL     string

	oplogAlwaysUpsert    *bool
	oplogApplicationMode *string
}

func buildOplogReplayRunArgs(cmdargs []string) (args oplogReplayRunArgs, err error) {
	// resolve archiving settings
	downloader, err := archive.NewStorageDownloader(archive.NewDefaultStorageSettings())
	if err != nil {
		return args, err
	}
	args.since, err = processArg(cmdargs[0], downloader)
	if err != nil {
		return
	}
	args.until, err = processArg(cmdargs[1], downloader)
	if err != nil {
		return
	}

	// TODO: fix ugly config
	if ignoreErrCodesStr, ok := internal.GetSetting(internal.OplogReplayIgnoreErrorCodes); ok {
		if err = json.Unmarshal([]byte(ignoreErrCodesStr), &args.ignoreErrCodes); err != nil {
			return
		}
	}

	args.mongodbURL, err = internal.GetRequiredSetting(internal.MongoDBUriSetting)
	if err != nil {
		return
	}

	oplogAlwaysUpsert, hasOplogAlwaysUpsert, err := internal.GetBoolSetting(internal.OplogReplayOplogAlwaysUpsert)
	if err != nil {
		return
	}
	if hasOplogAlwaysUpsert {
		args.oplogAlwaysUpsert = &oplogAlwaysUpsert
	}

	if oplogApplicationMode, hasOplogApplicationMode := internal.GetSetting(
		internal.OplogReplayOplogApplicationMode); hasOplogApplicationMode {
		args.oplogApplicationMode = &oplogApplicationMode
	}

	return args, nil
}

func processArg(arg string, downloader *archive.StorageDownloader) (models.Timestamp, error) {
	switch arg {
	case internal.LatestString:
		return downloader.LastKnownArchiveTS()
	case LatestBackupString:
		lastBackupName, err := downloader.LastBackupName()
		if err != nil {
			return models.Timestamp{}, err
		}
		backupMeta, err := downloader.BackupMeta(lastBackupName)
		if err != nil {
			return models.Timestamp{}, err
		}
		return models.TimestampFromBson(backupMeta.MongoMeta.BackupLastTS), nil
	default:
		return models.TimestampFromStr(arg)
	}
}

func runOplogReplay(ctx context.Context, replayArgs oplogReplayRunArgs) error {
	tracelog.DebugLogger.Printf("starting replay with arguments: %+v", replayArgs)

	// set up mongodb client and oplog applier
	var mongoClientArgs []client.Option
	if replayArgs.oplogAlwaysUpsert != nil {
		mongoClientArgs = append(mongoClientArgs, client.OplogAlwaysUpsert(*replayArgs.oplogAlwaysUpsert))
	}

	if replayArgs.oplogApplicationMode != nil {
		mongoClientArgs = append(mongoClientArgs,
			client.OplogApplicationMode(client.OplogAppMode(*replayArgs.oplogApplicationMode)))
	}

	mongoClient, err := client.NewMongoClient(ctx, replayArgs.mongodbURL, mongoClientArgs...)
	if err != nil {
		return err
	}

	if err = mongoClient.EnsureIsMaster(ctx); err != nil {
		return err
	}

	dbApplier := oplog.NewDBApplier(mongoClient, false, replayArgs.ignoreErrCodes)
	oplogApplier := stages.NewGenericApplier(dbApplier)

	// set up storage downloader client
	downloader, err := archive.NewStorageDownloader(archive.NewDefaultStorageSettings())
	if err != nil {
		return err
	}
	// discover archive sequence to replay
	archives, err := downloader.ListOplogArchives()
	if err != nil {
		return err
	}
	path, err := archive.SequenceBetweenTS(archives, replayArgs.since, replayArgs.until)
	if err != nil {
		return err
	}

	// setup storage fetcher
	oplogFetcher := stages.NewStorageFetcher(downloader, path)

	// run worker cycle
	return mongo.HandleOplogReplay(ctx, replayArgs.since, replayArgs.until, oplogFetcher, oplogApplier)
}

func init() {
	cmd.AddCommand(oplogReplayCmd)
}
