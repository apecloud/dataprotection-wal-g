package sqlserver

import (
	"context"
	"os"
	"syscall"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/sqlserver/blob"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

func RunProxy(folder storage.Folder) {
	ctx, cancel := context.WithCancel(context.Background())
	signalHandler := utility.NewSignalHandler(ctx, cancel, []os.Signal{syscall.SIGINT, syscall.SIGTERM})
	defer func() { _ = signalHandler.Close() }()
	bs, err := blob.NewServer(folder)
	tracelog.ErrorLogger.FatalfOnError("proxy create error: %v", err)
	lock, err := bs.AcquireLock()
	tracelog.ErrorLogger.FatalOnError(err)
	defer func() { tracelog.ErrorLogger.PrintOnError(lock.Close()) }()
	err = bs.Run(ctx)
	tracelog.ErrorLogger.FatalfOnError("proxy run error: %v", err)
}
