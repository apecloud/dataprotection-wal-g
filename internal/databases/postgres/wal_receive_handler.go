package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/apecloud/dataprotection-wal-g/internal"

	"github.com/apecloud/dataprotection-wal-g/internal/ioextensions"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/jackc/pgconn"
	"github.com/jackc/pglogrepl"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

const (
	// Sets standbyMessageTimeout in Streaming Replication Protocol.
	StandbyMessageTimeout = time.Second * 10
)

/*
NOTE: Preventing a WAL gap is a complex one (also not 100% fixed with arch_command).
* Using replication slot helps, but that should be created and maintained
  by wal-g on standby's too (making sure unconsumed wals are preserved on
  potential new masters too)
* Using sync replication is another option, but non-promotable, and we
  should locally cache to disconnect S3 performance from database performance
* Making something that checks 'what is in wal-g s repo' vs 'where postgres is
  is another option, but when wal-g is no longer running there would be nothing
  preventing postgres from advancing and cleaning, which is what slots are for.
Cleanest would probably be to create the slot on all postgres instances and advance all of them.
Can be done, but first, lets focus on creating wal files from repl msg...

Things to do (future):
* unittests for queryrunner code
* upgrade to pgx/v4
* we might want to add a feature to have wal-g advance multiple slots to support HA setups natively
* Test with different wal size (>=pg11)
*/

type genericWalReceiveError struct {
	error
}

func (err genericWalReceiveError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

// HandleWALReceive is invoked to receive wal with a replication connection and push
func HandleWALReceive(uploader *WalUploader) {
	// Connect to postgres.
	var XLogPos pglogrepl.LSN
	var segment *WalSegment

	uploader.ChangeDirectory(utility.WalPath)

	slot, walSegmentBytes, err := getCurrentWalInfo()
	tracelog.ErrorLogger.FatalOnError(err)
	tracelog.DebugLogger.Printf("WAL segment bytes: %d", walSegmentBytes)

	conn, err := pgconn.Connect(context.Background(), "replication=yes")
	tracelog.ErrorLogger.FatalOnError(err)
	defer conn.Close(context.Background())

	sysident, err := pglogrepl.IdentifySystem(context.Background(), conn)
	tracelog.ErrorLogger.FatalOnError(err)

	if slot.Exists {
		XLogPos = slot.RestartLSN
	} else {
		tracelog.InfoLogger.Println("Trying to create the replication slot")
		_, err = pglogrepl.CreateReplicationSlot(context.Background(), conn, slot.Name, "",
			pglogrepl.CreateReplicationSlotOptions{Mode: pglogrepl.PhysicalReplication})
		tracelog.ErrorLogger.FatalOnError(err)
		XLogPos = sysident.XLogPos
	}

	// Get timeline for XLogPos from historyfile with helper function
	timeline, err := getStartTimeline(conn, uploader, uint32(sysident.Timeline), XLogPos)
	tracelog.ErrorLogger.FatalOnError(err)

	segment = NewWalSegment(timeline, XLogPos, walSegmentBytes)
	startReplication(conn, segment, slot.Name)
	for {
		streamResult, err := segment.Stream(conn, StandbyMessageTimeout)
		tracelog.ErrorLogger.FatalOnError(err)
		tracelog.DebugLogger.Printf("Successfully received wal segment %s: ", segment.Name())

		switch streamResult {
		case ProcessMessageOK:
			// segment is a regular segemnt. Write, and create a new for this timeline.
			err = uploader.UploadWalFile(ioextensions.NewNamedReaderImpl(segment, segment.Name()))
			tracelog.ErrorLogger.FatalOnError(err)
			err = uploadRemoteWalMetadata(segment.Name(), uploader.Uploader)
			tracelog.ErrorLogger.FatalOnError(err)
			XLogPos = segment.endLSN
			segment, err = segment.NextWalSegment()
			tracelog.ErrorLogger.FatalOnError(err)
		case ProcessMessageCopyDone:
			// segment is a partial. Write, and create a new for the next timeline.
			err = uploader.UploadWalFile(ioextensions.NewNamedReaderImpl(segment, segment.Name()))
			tracelog.ErrorLogger.FatalOnError(err)
			err = uploadRemoteWalMetadata(segment.Name(), uploader.Uploader)
			tracelog.ErrorLogger.FatalOnError(err)
			timeline++
			timelinehistfile, err := pglogrepl.TimelineHistory(context.Background(), conn, int32(timeline))
			tracelog.ErrorLogger.FatalOnError(err)
			tlh, err := NewTimeLineHistFile(timeline, timelinehistfile.FileName, timelinehistfile.Content)
			tracelog.ErrorLogger.FatalOnError(err)
			err = uploader.UploadWalFile(ioextensions.NewNamedReaderImpl(tlh, tlh.Name()))
			tracelog.ErrorLogger.FatalOnError(err)
			err = uploadRemoteWalMetadata(tlh.Name(), uploader.Uploader)
			tracelog.ErrorLogger.FatalOnError(err)
			segment = NewWalSegment(timeline, XLogPos, walSegmentBytes)
			startReplication(conn, segment, slot.Name)
		default:
			tracelog.ErrorLogger.FatalOnError(errors.Errorf("Unexpected result from WalSegment.Stream() %v", streamResult))
		}
	}
}

func getStartTimeline(conn *pgconn.PgConn,
	uploader *WalUploader,
	systemTimeline uint32,
	xLogPos pglogrepl.LSN) (uint32, error) {
	if systemTimeline < 2 {
		return 1, nil
	}
	timelinehistfile, err := pglogrepl.TimelineHistory(context.Background(), conn, int32(systemTimeline))
	if err == nil {
		tlh, err := NewTimeLineHistFile(systemTimeline, timelinehistfile.FileName, timelinehistfile.Content)
		tracelog.ErrorLogger.FatalOnError(err)
		err = uploader.UploadWalFile(ioextensions.NewNamedReaderImpl(tlh, tlh.Name()))
		tracelog.ErrorLogger.FatalOnError(err)
		return tlh.LSNToTimeLine(xLogPos)
	}
	if pgErr, ok := err.(*pgconn.PgError); ok {
		if pgErr.Code == "58P01" {
			return systemTimeline, nil
		}
	}
	return 0, nil
}

func startReplication(conn *pgconn.PgConn, segment *WalSegment, slotName string) {
	tracelog.DebugLogger.Printf("Starting replication from %s: ", segment.StartLSN)
	err := pglogrepl.StartReplication(context.Background(), conn, slotName, segment.StartLSN,
		pglogrepl.StartReplicationOptions{Timeline: int32(segment.TimeLine), Mode: pglogrepl.PhysicalReplication})
	tracelog.ErrorLogger.FatalOnError(err)
	tracelog.DebugLogger.Println("Started replication")
}

func getCurrentWalInfo() (slot PhysicalSlot, walSegmentBytes uint64, err error) {
	slotName := internal.GetPgSlotName()

	// Creating a temporary connection to read slot info and wal_segment_size
	tmpConn, err := Connect()
	if err != nil {
		return
	}
	defer tmpConn.Close()

	queryRunner, err := NewPgQueryRunner(tmpConn)
	if err != nil {
		return
	}

	slot, err = queryRunner.GetPhysicalSlotInfo(slotName)
	if err != nil {
		return
	}

	walSegmentBytes, err = queryRunner.GetWalSegmentBytes()
	return
}
