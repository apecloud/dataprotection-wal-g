package postgres

import (
	"bytes"
	"io"
	"sort"

	"github.com/apecloud/dataprotection-wal-g/internal"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/jedib0t/go-pretty/table"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/wal-g/tracelog"
)

type IntegrityCheckDetails []*IntegrityScanSegmentSequence

func (sequences IntegrityCheckDetails) NewPlainTextReader() (io.Reader, error) {
	var outputBuffer bytes.Buffer

	tableWriter := table.NewWriter()
	tableWriter.SetOutputMirror(&outputBuffer)
	defer tableWriter.Render()

	tableWriter.AppendHeader(table.Row{"TLI", "Start", "End", "Segments count", "Status"})
	for _, row := range sequences {
		tableWriter.AppendRow(table.Row{row.TimelineID,
			row.StartSegment, row.EndSegment, row.SegmentsCount, row.Status})
	}

	return &outputBuffer, nil
}

// IntegrityCheckRunner queries the current cluster WAL segment and timeline
// and travels through WAL segments in storage in reversed chronological order (starting from that segment)
// to find any missing WAL segments that could potentially fail the PITR procedure
type IntegrityCheckRunner struct {
	startWalSegment           WalSegmentDescription
	stopWalSegmentNo          WalSegmentNo
	uploadingSegmentRangeSize int
	delayedSegmentRangeSize   int
	walFolderFilenames        []string
	timelineSwitchMap         map[WalSegmentNo]*TimelineHistoryRecord
	noBackupsFound            bool
}

func NewIntegrityCheckRunner(
	rootFolder storage.Folder,
	walFolderFilenames []string,
	currentWalSegment WalSegmentDescription,
) (IntegrityCheckRunner, error) {
	walFolder := rootFolder.GetSubFolder(utility.WalPath)

	timelineSwitchMap, err := createTimelineSwitchMap(currentWalSegment.Timeline, walFolder)
	if err != nil {
		return IntegrityCheckRunner{}, errors.Wrap(err, "Failed to initialize timeline history map")
	}

	noBackupsFound := false
	stopWalSegmentNo, err := getEarliestBackupStartSegmentNo(timelineSwitchMap, currentWalSegment.Timeline, rootFolder)
	if err != nil {
		tracelog.WarningLogger.Printf("Failed to detect earliest backup WAL segment no: '%v',"+
			"will scan until the 0000000X0000000000000001 segment.\n", err)
		stopWalSegmentNo = 1
		noBackupsFound = true
	}

	// uploadingSegmentRangeSize is needed to determine max amount of missing WAL segments
	// after the last found WAL segment which can be marked as "uploading"
	uploadingSegmentRangeSize, err := internal.GetMaxUploadConcurrency()
	if err != nil {
		return IntegrityCheckRunner{}, errors.Wrap(err, "Failed to resolve MaxUploadConcurrency")
	}

	return IntegrityCheckRunner{
		startWalSegment:           currentWalSegment,
		stopWalSegmentNo:          stopWalSegmentNo,
		uploadingSegmentRangeSize: uploadingSegmentRangeSize,
		delayedSegmentRangeSize:   viper.GetInt(internal.MaxDelayedSegmentsCount),
		walFolderFilenames:        walFolderFilenames,
		timelineSwitchMap:         timelineSwitchMap,
		noBackupsFound:            noBackupsFound,
	}, nil
}

func (check IntegrityCheckRunner) Run() (WalVerifyCheckResult, error) {
	storageSegments := getSegmentsFromFiles(check.walFolderFilenames)
	walSegmentRunner := NewWalSegmentRunner(check.startWalSegment,
		storageSegments,
		check.stopWalSegmentNo,
		check.timelineSwitchMap)

	segmentScanner := NewWalSegmentScanner(walSegmentRunner)
	err := runWalIntegrityScan(segmentScanner, check.uploadingSegmentRangeSize, check.delayedSegmentRangeSize)
	if err != nil {
		return WalVerifyCheckResult{}, err
	}

	integrityScanSegmentSequences := collapseSegmentsByStatusAndTimeline(segmentScanner.ScannedSegments)

	return check.newWalIntegrityCheckResult(integrityScanSegmentSequences), nil
}

func (check IntegrityCheckRunner) Type() WalVerifyCheckType {
	return WalVerifyIntegrityCheck
}

// newWalIntegrityCheckResult check produces the WalVerifyCheckResult with status:
// StatusOk if there are no missing segments in storage
// StatusWarning if storage contains some ProbablyUploading or ProbablyDelayed segments
// StatusFailure if storage contains Lost segments
func (check IntegrityCheckRunner) newWalIntegrityCheckResult(segmentSequences []*IntegrityScanSegmentSequence) WalVerifyCheckResult {
	result := WalVerifyCheckResult{
		Status:  StatusOk,
		Details: IntegrityCheckDetails(segmentSequences),
	}

	prevFound := false
	for _, row := range segmentSequences {
		switch row.Status {
		case Found:
			prevFound = true
		case Lost:
			if check.noBackupsFound && !prevFound {
				// If there are no backups in storage, WAL-G can't determine the first segment to start the scan from.
				// So, skip the first not found segment sequences instead of failing.
				result.Status = StatusWarning
			} else {
				result.Status = StatusFailure
				return result
			}
		case ProbablyDelayed, ProbablyUploading:
			result.Status = StatusWarning
		}
	}
	return result
}

// IntegrityScanSegmentSequence is a continuous sequence of segments
// with the same timeline and Status
type IntegrityScanSegmentSequence struct {
	TimelineID    uint32               `json:"timeline_id"`
	StartSegment  string               `json:"start_segment"`
	EndSegment    string               `json:"end_segment"`
	SegmentsCount int                  `json:"segments_count"`
	Status        ScannedSegmentStatus `json:"status"`
}

func newIntegrityScanSegmentSequence(sequence *WalSegmentsSequence,
	status ScannedSegmentStatus) *IntegrityScanSegmentSequence {
	return &IntegrityScanSegmentSequence{
		TimelineID:    sequence.TimelineID,
		StartSegment:  sequence.MinSegmentNo.getFilename(sequence.TimelineID),
		EndSegment:    sequence.MaxSegmentNo.getFilename(sequence.TimelineID),
		Status:        status,
		SegmentsCount: len(sequence.WalSegmentNumbers),
	}
}

// runWalIntegrityScan invokes the following storage scan series
// (on each iteration scanner continues from the position where it stopped)
// 1. At first, it runs scan until it finds some segment in WAL storage
// and marks all encountered missing segments as "missing, probably delayed"
// 2. Then it scans exactly uploadingSegmentRangeSize count of segments,
// if found any missing segments it marks them as "missing, probably still uploading"
// 3. Final scan without any limit (until stopSegment is reached),
// all missing segments encountered in this scan are considered as "missing, lost"
func runWalIntegrityScan(scanner *WalSegmentScanner,
	uploadingSegmentRangeSize, delayedSegmentRangeSize int) error {
	// Run to the latest WAL segment available in storage, mark all missing segments as delayed
	err := scanner.Scan(SegmentScanConfig{
		ScanSegmentsLimit:       delayedSegmentRangeSize,
		StopOnFirstFoundSegment: true,
		MissingSegmentStatus:    ProbablyDelayed,
	})
	if err != nil {
		return err
	}

	// Traverse potentially uploading segments, mark all missing segments as probably uploading
	err = scanner.Scan(SegmentScanConfig{
		ScanSegmentsLimit:    uploadingSegmentRangeSize,
		MissingSegmentStatus: ProbablyUploading,
	})
	if err != nil {
		return err
	}

	// Run until stop segment, and mark all missing segments as lost
	return scanner.Scan(SegmentScanConfig{
		UnlimitedScan:        true,
		MissingSegmentStatus: Lost,
	})
}

// collapseSegmentsByStatusAndTimeline collapses scanned segments
// with the same timeline and Status into segment sequences to minify the output
func collapseSegmentsByStatusAndTimeline(scannedSegments []ScannedSegmentDescription) []*IntegrityScanSegmentSequence {
	if len(scannedSegments) == 0 {
		return nil
	}

	// make sure that ScannedSegments are ordered
	sort.Slice(scannedSegments, func(i, j int) bool {
		return scannedSegments[i].Number < scannedSegments[j].Number
	})

	segmentSequences := make([]*IntegrityScanSegmentSequence, 0)
	currentSequence := NewSegmentsSequence(scannedSegments[0].Timeline, scannedSegments[0].Number)
	currentStatus := scannedSegments[0].status

	for i := 1; i < len(scannedSegments); i++ {
		segment := scannedSegments[i]

		// switch to the new sequence on segment Status change or timeline id change
		if segment.status != currentStatus || currentSequence.TimelineID != segment.Timeline {
			segmentSequences = append(segmentSequences, newIntegrityScanSegmentSequence(currentSequence, currentStatus))
			currentSequence = NewSegmentsSequence(segment.Timeline, segment.Number)
			currentStatus = segment.status
		} else {
			currentSequence.AddWalSegmentNo(segment.Number)
		}
	}

	segmentSequences = append(segmentSequences, newIntegrityScanSegmentSequence(currentSequence, currentStatus))
	return segmentSequences
}

// getEarliestBackupStartSegmentNo returns the starting segmentNo of the earliest available correct backup
func getEarliestBackupStartSegmentNo(timelineSwitchMap map[WalSegmentNo]*TimelineHistoryRecord,
	currentTimeline uint32,
	rootFolder storage.Folder) (WalSegmentNo, error) {
	backups, err := internal.GetBackups(rootFolder.GetSubFolder(utility.BaseBackupPath))
	if err != nil {
		return 0, err
	}

	backupDetails, err := GetBackupsDetails(rootFolder.GetSubFolder(utility.BaseBackupPath), backups)
	if err != nil {
		return 0, err
	}

	// switchLsnBySegNo is used for fast lookup of the timeline switch segment
	switchLsnBySegNo := make(map[uint32]WalSegmentNo, len(timelineSwitchMap))
	for _, historyRecord := range timelineSwitchMap {
		switchLsnBySegNo[historyRecord.timeline] = newWalSegmentNo(historyRecord.lsn)
	}
	earliestBackup, earliestBackupSegNo, err :=
		findEarliestBackup(currentTimeline, backupDetails, switchLsnBySegNo)
	if err != nil {
		return 0, err
	}

	tracelog.InfoLogger.Printf("Detected earliest available backup: %s\n",
		earliestBackup.BackupName)
	return earliestBackupSegNo, nil
}

// findEarliestBackup finds earliest correct backup available in storage.
func findEarliestBackup(
	currentTimeline uint32,
	backupDetails []BackupDetail,
	switchSegNoByTimeline map[uint32]WalSegmentNo,
) (*BackupDetail, WalSegmentNo, error) {
	var earliestBackup *BackupDetail
	var earliestBackupSegNo WalSegmentNo

	for i := range backupDetails {
		currBackup := &backupDetails[i]
		backupTimelineID, backupLogSegNoInt, err := ParseWALFilename(currBackup.WalFileName)
		backupLogSegNo := WalSegmentNo(backupLogSegNoInt)
		if err != nil {
			return nil, 0, err
		}

		if ok := checkBackupIsCorrect(currentTimeline, currBackup,
			backupTimelineID, backupLogSegNo, switchSegNoByTimeline); !ok {
			continue
		}

		if earliestBackup == nil || earliestBackupSegNo > backupLogSegNo {
			earliestBackupSegNo = backupLogSegNo
			earliestBackup = currBackup
		}
	}
	if earliestBackup == nil {
		return nil, 0, newNoCorrectBackupFoundError()
	}
	return earliestBackup, earliestBackupSegNo, nil
}

// checkBackupIsCorrect checks that:
// 1. backup start LSN is valid
// 2. backup is not permanent
//
// Backup start LSN is considered valid if
// it belongs to range [backup timeline start LSN, backup timeline end LSN]
func checkBackupIsCorrect(
	currentTimeline uint32,
	backupDetail *BackupDetail,
	backupTimeline uint32,
	backupStartSegNo WalSegmentNo,
	switchSegNoByTimeline map[uint32]WalSegmentNo,
) bool {
	// if backup is permanent, it is not eligible for wal-verify to be selected as the left border
	if backupDetail.IsPermanent {
		tracelog.WarningLogger.Printf(
			"checkBackupIsCorrect: %s: backup is permanent, it is not eligible to be selected "+
				"as the earliest backup for wal-verify.\n", backupDetail.BackupName)
		return false
	}

	// perform the check only if .history file exists
	if len(switchSegNoByTimeline) > 0 {
		// if backup start segment is less than timeline start segment => incorrect backup
		if backupTimeline > 1 {
			backupTimelineStartSegNo, ok := switchSegNoByTimeline[backupTimeline-1]
			if ok && backupStartSegNo < backupTimelineStartSegNo {
				tracelog.WarningLogger.Printf(
					"checkBackupIsCorrect: %s: backup start segment number %d "+
						"is less than the backup timeline start segment number %d.\n",
					backupDetail.BackupName, backupStartSegNo, backupTimelineStartSegNo)
				return false
			}
		}

		// if backup belongs to the current timeline, skip the rest of the checks
		if backupTimeline == currentTimeline {
			return true
		}

		// if backup timeline is not present in current .history file => incorrect backup
		timelineSwitchSegNo, ok := switchSegNoByTimeline[backupTimeline]
		if !ok {
			tracelog.WarningLogger.Printf(
				"checkBackupIsCorrect: %s: backup timeline %d "+
					"is not present in .history file and is not current.\n",
				backupDetail.BackupName, backupTimeline)
			return false
		}

		// if backup start segment is higher than switch segment of the previous timeline => incorrect backup
		if backupStartSegNo >= timelineSwitchSegNo {
			tracelog.WarningLogger.Printf(
				"checkBackupIsCorrect: %s: backup start segment number %d "+
					"should be less than the backup timeline end segment number %d.\n",
				backupDetail.BackupName, backupStartSegNo, timelineSwitchSegNo)
			return false
		}
	}
	return true
}
