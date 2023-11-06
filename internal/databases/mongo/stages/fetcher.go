package stages

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/wal-g/tracelog"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/archive"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/client"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/models"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/stages/buffer"

	"go.mongodb.org/mongo-driver/bson"
)

var (
	_ = []GapHandler{&StorageGapHandler{}}
	_ = []Fetcher{&CursorMajFetcher{}}
	_ = []BetweenFetcher{&StorageFetcher{}}
)

const OperationNoop = "\"n\""

type GapHandler interface {
	HandleGap(from, until models.Timestamp, err error) error
}

type StorageGapHandler struct {
	uploader archive.Uploader
}

func NewStorageGapHandler(uploader archive.Uploader) *StorageGapHandler {
	return &StorageGapHandler{uploader}
}

func (sgh *StorageGapHandler) HandleGap(from, until models.Timestamp, gapErr error) error {
	if err := sgh.uploader.UploadGapArchive(gapErr, from, until); err != nil {
		return fmt.Errorf("can not upload gap archive: %w", err)
	}
	return nil
}

// Fetcher defines interface to fetch oplog records.
// TODO: FIX INTERFACE METHOD NAME AND SIGNATURE
type Fetcher interface {
	Fetch(context.Context) (chan *models.Oplog, chan error, error)
}

// BetweenFetcher defines interface to fetch oplog records between given timestamps.
type BetweenFetcher interface {
	FetchBetween(context.Context, models.Timestamp, models.Timestamp) (chan *models.Oplog, chan error, error)
}

// CursorMajFetcher implements Fetcher interface for mongodb
type CursorMajFetcher struct {
	db         client.MongoDriver
	cur        client.OplogCursor
	lwInterval time.Duration
}

// NewCursorMajFetcher builds CursorMajFetcher with given args.
func NewCursorMajFetcher(m client.MongoDriver,
	cur client.OplogCursor,
	lwUpdateInterval time.Duration) *CursorMajFetcher {
	return &CursorMajFetcher{m, cur, lwUpdateInterval}
}

func (dbf *CursorMajFetcher) isNoopOp(raw bson.Raw) bool {
	return raw.Lookup("op").String() == OperationNoop
}

// Fetch returns channel of oplog records, channel is filled in background.
// TODO: use sessions
// TODO: use context.WithTimeout
func (dbf *CursorMajFetcher) Fetch(ctx context.Context) (oplogc chan *models.Oplog, errc chan error, err error) {
	oplogc = make(chan *models.Oplog)
	errc = make(chan error)
	go func() {
		defer close(errc)
		defer close(oplogc)

		majTS := models.Timestamp{}
		for dbf.cur.Next(ctx) {
			// TODO: benchmark decode vs. bson.Reader vs. bson.Raw.LookupErr
			op, err := models.OplogFromRaw(dbf.cur.Data())
			if err != nil {
				errc <- fmt.Errorf("oplog record decoding failed: %w", err)
				return
			}
			if dbf.isNoopOp(op.Data) {
				continue
			}
			// TODO: move to separate component and fetch last writes in background
			for models.LessTS(majTS, op.TS) {
				time.Sleep(dbf.lwInterval)

				im, err := dbf.db.IsMaster(ctx)
				if err != nil {
					errc <- err
					return
				}

				// TODO: support archiving from secondary
				if !im.IsMaster {
					errc <- fmt.Errorf("current node is not a primary")
					return
				}

				majTS = im.LastWrite.MajorityOpTime.TS
			}

			select {
			case oplogc <- op:
			case <-ctx.Done():
				return
			}
		}

		if err := dbf.cur.Err(); err != nil {
			if err == ctx.Err() {
				return
			}
			errc <- fmt.Errorf("oplog cursor error: %w", err)
			return
		}
		errc <- fmt.Errorf("oplog cursor exhausted")
	}()

	return oplogc, errc, nil
}

// StorageFetcher implements BetweenFetcher interface for storage.
type StorageFetcher struct {
	downloader archive.Downloader
	path       archive.Sequence
}

// NewStorageFetcher builds StorageFetcher instance
func NewStorageFetcher(downloader archive.Downloader, path archive.Sequence) *StorageFetcher {
	return &StorageFetcher{downloader: downloader, path: path}
}

// FetchBetween returns channel of oplog records, channel is filled in background.
func (sf *StorageFetcher) FetchBetween(ctx context.Context,
	from,
	until models.Timestamp) (oplogc chan *models.Oplog, errc chan error, err error) {
	if models.LessTS(until, from) {
		return nil, nil, fmt.Errorf("fromTS '%s' must be less than untilTS '%s'", from, until)
	}

	data := make(chan *models.Oplog)
	errc = make(chan error)
	go func() {
		defer close(errc)
		defer close(data)

		buf := buffer.NewCloserBuffer() // TODO: switch to streaming interface
		path := sf.path
		firstFound := false

		for _, arch := range path {
			tracelog.DebugLogger.Printf("Fetching archive %s", arch.Filename())

			err := sf.downloader.DownloadOplogArchive(arch, buf)
			if err != nil {
				errc <- fmt.Errorf("failed to download archive %s: %w", arch.Filename(), err)
				return
			}

			for {
				// TODO: benchmark & compare with bson_stream
				raw, err := bson.NewFromIOReader(buf)
				if err != nil {
					if err == io.EOF {
						break
					}
					errc <- fmt.Errorf("error during read bson: %w", err)
				}

				op, err := models.OplogFromRaw(raw)
				if err != nil {
					errc <- fmt.Errorf("oplog record decoding failed: %w", err)
					return
				}

				if !firstFound {
					// op.TS less than from, continue
					if models.LessTS(op.TS, from) {
						continue
					}
					firstFound = true
				}

				// TODO: do we need also check every op "op.TS > from"
				if models.LessTS(until, op.TS) || op.TS == until {
					tracelog.InfoLogger.Println("Oplog archives fetching is completed")
					return
				}

				// tracelog.DebugLogger.Printf("Fetcher receieved op %s (%s on %s)", op.TS, op.OP, op.NS)
				select {
				case data <- op:
				case <-ctx.Done():
					tracelog.InfoLogger.Println("Oplog archives fetching is canceled")
					return
				}
			}
			buf.Reset()
			if !firstFound { // TODO: do we need this check, add skip flag
				errc <- fmt.Errorf("'from' timestamp '%s' was not found in first archive: %s", from, arch.Filename())
				return
			}
		}
		errc <- fmt.Errorf("restore sequence was fetched, but restore point '%s' is not reached",
			until)
	}()

	return data, errc, nil
}
