package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"syscall"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

func HandleBackupRestore(backupName string, dbnames []string, fromnames []string, noRecovery bool) {
	ctx, cancel := context.WithCancel(context.Background())
	signalHandler := utility.NewSignalHandler(ctx, cancel, []os.Signal{syscall.SIGINT, syscall.SIGTERM})
	defer func() { _ = signalHandler.Close() }()

	folder, err := internal.ConfigureFolder()
	tracelog.ErrorLogger.FatalOnError(err)

	backup, err := internal.GetBackupByName(backupName, utility.BaseBackupPath, folder)
	tracelog.ErrorLogger.FatalOnError(err)

	sentinel := new(SentinelDto)
	err = backup.FetchSentinel(sentinel)
	tracelog.ErrorLogger.FatalOnError(err)

	db, err := getSQLServerConnection()
	tracelog.ErrorLogger.FatalfOnError("failed to connect to SQLServer: %v", err)

	dbnames, fromnames, err = getDatabasesToRestore(sentinel, dbnames, fromnames)
	tracelog.ErrorLogger.FatalfOnError("failed to list databases to restore: %v", err)

	lock, err := RunOrReuseProxy(ctx, cancel, folder)
	tracelog.ErrorLogger.FatalOnError(err)
	defer lock.Close()

	backupName = backup.Name

	err = runParallel(func(i int) error {
		dbname := dbnames[i]
		fromname := fromnames[i]
		err := restoreSingleDatabase(ctx, db, folder, backupName, dbname, fromname)
		if err != nil {
			return err
		}
		if !noRecovery {
			return recoverSingleDatabase(ctx, db, dbname)
		}
		return nil
	}, len(dbnames), getDBConcurrency())
	tracelog.ErrorLogger.FatalfOnError("overall restore failed: %v", err)

	tracelog.InfoLogger.Printf("restore finished")
}

func restoreSingleDatabase(ctx context.Context,
	db *sql.DB,
	folder storage.Folder,
	backupName string,
	dbname string,
	fromName string) error {
	baseURL := getDatabaseBackupURL(backupName, fromName)
	basePath := getDatabaseBackupPath(backupName, fromName)
	blobs, err := listBackupBlobs(folder.GetSubFolder(basePath))
	if err != nil {
		return err
	}
	urls := buildRestoreUrls(baseURL, blobs)
	sql := fmt.Sprintf("RESTORE DATABASE %s FROM %s WITH REPLACE, NORECOVERY", quoteName(dbname), urls)
	files, err := listDatabaseFiles(db, urls)
	if err != nil {
		return err
	}
	datadir, logdir, err := GetDefaultDataLogDirs(db)
	if err != nil {
		return err
	}
	move, err := buildPhysicalFileMove(files, dbname, datadir, logdir)
	if err != nil {
		return err
	}
	sql += ", " + move
	tracelog.InfoLogger.Printf("starting restore database [%s] from %s", dbname, urls)
	tracelog.DebugLogger.Printf("SQL: %s", sql)
	_, err = db.ExecContext(ctx, sql)
	if err != nil {
		tracelog.ErrorLogger.Printf("database [%s] restore failed: %v", dbname, err)
	} else {
		tracelog.InfoLogger.Printf("database [%s] restore succefully finished", dbname)
	}
	return err
}

func recoverSingleDatabase(ctx context.Context, db *sql.DB, dbname string) error {
	sql := fmt.Sprintf("RESTORE DATABASE %s WITH RECOVERY", quoteName(dbname))
	tracelog.InfoLogger.Printf("recovering database [%s]", dbname)
	tracelog.DebugLogger.Printf("SQL: %s", sql)
	_, err := db.ExecContext(ctx, sql)
	if err != nil {
		tracelog.ErrorLogger.Printf("database [%s] recovery failed: %v", dbname, err)
	} else {
		tracelog.InfoLogger.Printf("database [%s] recovery succefully finished", dbname)
	}
	return err
}
