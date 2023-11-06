package postgres_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func init() {
	// this setting affects the ProbablyUploading segments range size
	viper.Set(internal.UploadConcurrencySetting, "4")

	// this setting controls the ProbablyDelayed segments range size
	viper.Set(internal.MaxDelayedSegmentsCount, "3")
}

type WalVerifyTestSetup struct {
	expectedIntegrityCheck postgres.WalVerifyCheckResult
	expectedTimelineCheck  postgres.WalVerifyCheckResult

	// currentWalSegment represents the current cluster wal segment
	currentWalSegment postgres.WalSegmentDescription
	// list of mock storage wal folder WAL segments
	storageSegments []string
	// list of other mock storage files
	storageFiles map[string]*bytes.Buffer
}

// MockWalVerifyOutputWriter is used to capture wal-verify command output
type MockWalVerifyOutputWriter struct {
	lastResult map[postgres.WalVerifyCheckType]postgres.WalVerifyCheckResult
	// number of time Write() function has been called
	writeCallsCount int
}

func (writer *MockWalVerifyOutputWriter) Write(
	result map[postgres.WalVerifyCheckType]postgres.WalVerifyCheckResult) error {
	writer.lastResult = result
	writer.writeCallsCount += 1
	return nil
}

// test that wal-verify works correctly on empty storage
func TestWalVerify_EmptyStorage(t *testing.T) {
	currentSegmentName := "00000003000000000000000A"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	storageFiles := make(map[string]*bytes.Buffer)
	storageSegments := make([]string, 0)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusWarning,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    3,
				StartSegment:  "000000030000000000000001",
				EndSegment:    "000000030000000000000002",
				SegmentsCount: 2,
				Status:        postgres.Lost,
			},
			{
				TimelineID:    3,
				StartSegment:  "000000030000000000000003",
				EndSegment:    "000000030000000000000006",
				SegmentsCount: 4, //uploadingSegmentRangeSize
				Status:        postgres.ProbablyUploading,
			},
			{
				TimelineID:    3,
				StartSegment:  "000000030000000000000007",
				EndSegment:    "000000030000000000000009",
				SegmentsCount: 3, //delayedSegmentRangeSize
				Status:        postgres.ProbablyDelayed,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusWarning,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID:        currentSegment.Timeline,
			HighestStorageTimelineID: 0,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageFiles:           storageFiles,
		storageSegments:        storageSegments,
	})
}

// check that storage garbage doesn't affect the wal-verify command
func TestWalVerify_OnlyGarbageInStorage(t *testing.T) {
	storageSegments := []string{
		"00000007000000000000000K",
		"0000000Y000000000000000K",
	}

	storageFiles := map[string]*bytes.Buffer{
		"some_garbage_file": new(bytes.Buffer),
		" ":                 new(bytes.Buffer),
	}

	currentSegmentName := "00000003000000000000000A"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusWarning,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    3,
				StartSegment:  "000000030000000000000001",
				EndSegment:    "000000030000000000000002",
				SegmentsCount: 2,
				Status:        postgres.Lost,
			},
			{
				TimelineID:    3,
				StartSegment:  "000000030000000000000003",
				EndSegment:    "000000030000000000000006",
				SegmentsCount: 4, //uploadingSegmentRangeSize
				Status:        postgres.ProbablyUploading,
			},
			{
				TimelineID:    3,
				StartSegment:  "000000030000000000000007",
				EndSegment:    "000000030000000000000009",
				SegmentsCount: 3, //delayedSegmentRangeSize
				Status:        postgres.ProbablyDelayed,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusWarning,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID: currentSegment.Timeline,
			// WAL storage folder is empty so highest found timeline should be zero
			HighestStorageTimelineID: 0,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageFiles:           storageFiles,
		storageSegments:        storageSegments,
	})
}

// check that wal-verify works for single timeline
func TestWalVerify_SingleTimeline_Ok(t *testing.T) {
	storageSegments := []string{
		"000000050000000000000001",
		"000000050000000000000002",
		"000000050000000000000003",
		"000000050000000000000004",
	}
	currentSegmentName := "000000050000000000000005"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000001",
				EndSegment:    "000000050000000000000004",
				SegmentsCount: 4,
				Status:        postgres.Found,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID:        currentSegment.Timeline,
			HighestStorageTimelineID: currentSegment.Timeline,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageSegments:        storageSegments,
		storageFiles:           make(map[string]*bytes.Buffer),
	})
}

// check that wal-verify correctly marks delayed segments
func TestWalVerify_SingleTimeline_SomeDelayed(t *testing.T) {
	storageSegments := []string{
		"000000050000000000000001",
		"000000050000000000000002",
		"000000050000000000000003",
		"000000050000000000000004",
	}

	currentSegmentName := "000000050000000000000008"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusWarning,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000001",
				EndSegment:    "000000050000000000000004",
				SegmentsCount: 4,
				Status:        postgres.Found,
			},
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000005",
				EndSegment:    "000000050000000000000007",
				SegmentsCount: 3,
				Status:        postgres.ProbablyDelayed,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID:        currentSegment.Timeline,
			HighestStorageTimelineID: currentSegment.Timeline,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageSegments:        storageSegments,
		storageFiles:           make(map[string]*bytes.Buffer),
	})
}

// check that wal-verify correctly marks uploading segments
func TestWalVerify_SingleTimeline_SomeUploading(t *testing.T) {
	storageSegments := []string{
		"000000050000000000000001",
		"000000050000000000000002",
		"000000050000000000000003",
		"000000050000000000000005",
		"000000050000000000000007",
	}

	currentSegmentName := "000000050000000000000008"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusWarning,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000001",
				EndSegment:    "000000050000000000000003",
				SegmentsCount: 3,
				Status:        postgres.Found,
			},
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000004",
				EndSegment:    "000000050000000000000004",
				SegmentsCount: 1,
				Status:        postgres.ProbablyUploading,
			},
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000005",
				EndSegment:    "000000050000000000000005",
				SegmentsCount: 1,
				Status:        postgres.Found,
			},
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000006",
				EndSegment:    "000000050000000000000006",
				SegmentsCount: 1,
				Status:        postgres.ProbablyUploading,
			},
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000007",
				EndSegment:    "000000050000000000000007",
				SegmentsCount: 1,
				Status:        postgres.Found,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID:        currentSegment.Timeline,
			HighestStorageTimelineID: currentSegment.Timeline,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageSegments:        storageSegments,
		storageFiles:           make(map[string]*bytes.Buffer),
	})
}

// check that wal-verify correctly follows timeline switches
func TestWalVerify_TwoTimelines_Ok(t *testing.T) {
	storageSegments := []string{
		"000000050000000000000001",
		"000000050000000000000002",
		"000000050000000000000003",
		"000000050000000000000004",

		// These segments should be ignored
		// because they have higher LSN
		// than timeline switch LSN
		// which is specified in the .history file below

		"000000050000000000000005", // should be ignored
		"000000050000000000000006", // should be ignored
		"000000050000000000000007", // should be ignored
		"000000050000000000000008", // should be ignored
		"000000050000000000000009", // should be ignored

		"000000060000000000000005",
		"000000060000000000000006",
		"000000060000000000000007",
		"000000060000000000000008",
	}

	// set switch point to somewhere in the 5th segment
	switchPointLsn := 5*postgres.WalSegmentSize + 100
	historyContents := fmt.Sprintf("%d\t0/%X\tsome comment...\n\n", 5, switchPointLsn)
	historyName, historyFile, err := newTimelineHistoryFile(historyContents, 6)
	// .history file should be stored in wal folder
	historyName = utility.WalPath + historyName
	assert.NoError(t, err)

	currentSegmentName := "000000060000000000000009"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000001",
				EndSegment:    "000000050000000000000004",
				SegmentsCount: 4,
				Status:        postgres.Found,
			},
			{
				TimelineID:    6,
				StartSegment:  "000000060000000000000005",
				EndSegment:    "000000060000000000000008",
				SegmentsCount: 4,
				Status:        postgres.Found,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID:        currentSegment.Timeline,
			HighestStorageTimelineID: currentSegment.Timeline,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageSegments:        storageSegments,
		storageFiles:           map[string]*bytes.Buffer{historyName: historyFile},
	})
}

// check that wal-verify correctly reports Lost segments
func TestWalVerify_TwoTimelines_SomeLost(t *testing.T) {
	storageSegments := []string{
		"000000050000000000000001",
		"000000050000000000000002",
		"000000050000000000000004",
		"000000050000000000000005",
		"000000050000000000000006",
		"000000060000000000000007",
		"000000060000000000000008",
	}

	// set switch point to somewhere in the 5th segment
	switchPointLsn := 5*postgres.WalSegmentSize + 100
	historyContents := fmt.Sprintf("%d\t0/%X\tsome comment...\n\n", 5, switchPointLsn)
	historyName, historyFile, err := newTimelineHistoryFile(historyContents, 6)
	// .history file should be stored in wal folder
	historyName = utility.WalPath + historyName
	assert.NoError(t, err)

	currentSegmentName := "000000060000000000000009"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusFailure,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000001",
				EndSegment:    "000000050000000000000002",
				SegmentsCount: 2,
				Status:        postgres.Found,
			},
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000003",
				EndSegment:    "000000050000000000000003",
				SegmentsCount: 1,
				Status:        postgres.Lost,
			},
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000004",
				EndSegment:    "000000050000000000000004",
				SegmentsCount: 1,
				Status:        postgres.Found,
			},
			{
				TimelineID:    6,
				StartSegment:  "000000060000000000000005",
				EndSegment:    "000000060000000000000006",
				SegmentsCount: 2,
				Status:        postgres.ProbablyUploading,
			},
			{
				TimelineID:    6,
				StartSegment:  "000000060000000000000007",
				EndSegment:    "000000060000000000000008",
				SegmentsCount: 2,
				Status:        postgres.Found,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID:        currentSegment.Timeline,
			HighestStorageTimelineID: currentSegment.Timeline,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageSegments:        storageSegments,
		storageFiles:           map[string]*bytes.Buffer{historyName: historyFile},
	})
}

// wal-verify timeline check test
func TestWalVerify_HigherTimelineExists(t *testing.T) {
	storageSegments := []string{
		"000000050000000000000001",
		"000000050000000000000002",
		"000000050000000000000003",
		"000000050000000000000004",
		"000000070000000000000003",
		"000000070000000000000004",
	}
	currentSegmentName := "000000050000000000000005"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    5,
				StartSegment:  "000000050000000000000001",
				EndSegment:    "000000050000000000000004",
				SegmentsCount: 4,
				Status:        postgres.Found,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusFailure,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID:        currentSegment.Timeline,
			HighestStorageTimelineID: 7,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageSegments:        storageSegments,
		storageFiles:           make(map[string]*bytes.Buffer),
	})
}

// Check that correct backup is chosen for wal-verify range start
func TestWalVerify_WalkUntilFirstBackup(t *testing.T) {
	storageSegments := []string{
		"000000050000000000000001",
		"000000050000000000000002",
		"000000050000000000000003",
		"000000050000000000000004",

		// These segments should be ignored
		// because they have higher LSN
		// than timeline switch LSN
		// which is specified in the .history file below

		"000000050000000000000005", // should be ignored
		"000000050000000000000006", // should be ignored
		"000000050000000000000007", // should be ignored
		"000000050000000000000008", // should be ignored
		"000000050000000000000009", // should be ignored

		"000000060000000000000005",
		"000000060000000000000006",
		"000000060000000000000007",
		"000000060000000000000008",
	}

	storageFiles := make(map[string]*bytes.Buffer, 4)

	// there are a couple of mock backups in storage, but only one
	// should be selected as the first one
	backupsWalNames := map[string]postgres.ExtendedMetadataDto{
		// INCORRECT: this backup should not be selected as the earliest,
		// because it does not belong to the current timeline history
		// since the timeline switch occurred
		// at the 000000060000000000000005 WAL segment
		"000000050000000000000005": newMockExtendedMetadataDto(false),
		// INCORRECT: this backup should not be selected as the earliest, because it is not the earliest one
		"000000060000000000000007": newMockExtendedMetadataDto(false),
		// OK: this backup should be selected as the earliest
		"000000060000000000000006": newMockExtendedMetadataDto(false),
		// INCORRECT: backup has been created before the timeline switch LSN
		"000000060000000000000002": newMockExtendedMetadataDto(false),
		// INCORRECT: backup is marked permanent
		"000000050000000000000003": newMockExtendedMetadataDto(true),
	}

	addMockBackupsStorageFiles(backupsWalNames, storageFiles)

	// set switch point to somewhere in the 5th segment
	switchPointLsn := 5*postgres.WalSegmentSize + 100
	historyInfo := fmt.Sprintf("%d\t0/%X\tsome comment...\n\n", 5, switchPointLsn)
	historyName, historyFile, err := newTimelineHistoryFile(historyInfo, 6)
	// .history file should be stored in wal folder
	historyName = utility.WalPath + historyName
	assert.NoError(t, err)

	storageFiles[historyName] = historyFile

	currentSegmentName := "000000060000000000000009"
	currentSegment, _ := postgres.NewWalSegmentDescription(currentSegmentName)

	expectedIntegrityCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.IntegrityCheckDetails{
			{
				TimelineID:    6,
				StartSegment:  "000000060000000000000006",
				EndSegment:    "000000060000000000000008",
				SegmentsCount: 3,
				Status:        postgres.Found,
			},
		},
	}

	expectedTimelineCheck := postgres.WalVerifyCheckResult{
		Status: postgres.StatusOk,
		Details: postgres.TimelineCheckDetails{
			CurrentTimelineID:        currentSegment.Timeline,
			HighestStorageTimelineID: currentSegment.Timeline,
		},
	}

	testWalVerify(t, WalVerifyTestSetup{
		expectedIntegrityCheck: expectedIntegrityCheck,
		expectedTimelineCheck:  expectedTimelineCheck,
		currentWalSegment:      currentSegment,
		storageSegments:        storageSegments,
		storageFiles:           storageFiles,
	})
}

func addMockBackupsStorageFiles(backups map[string]postgres.ExtendedMetadataDto, storageFiles map[string]*bytes.Buffer) {
	for name, meta := range backups {
		// sentinel
		storageFiles[utility.BaseBackupPath+utility.BackupNamePrefix+name+utility.SentinelSuffix] = new(bytes.Buffer)
		// metadata
		metaBytes, _ := json.Marshal(meta)
		storageFiles[utility.BaseBackupPath+utility.BackupNamePrefix+name+"/"+utility.MetadataFileName] = bytes.NewBuffer(metaBytes)
	}
}

func testWalVerify(t *testing.T, setup WalVerifyTestSetup) {
	expectedResult := map[postgres.WalVerifyCheckType]postgres.WalVerifyCheckResult{
		postgres.WalVerifyTimelineCheck:  setup.expectedTimelineCheck,
		postgres.WalVerifyIntegrityCheck: setup.expectedIntegrityCheck,
	}

	result, outputCallsCount := executeWalVerify(
		setup.storageSegments,
		setup.storageFiles,
		setup.currentWalSegment)

	assert.Equal(t, 1, outputCallsCount)
	compareResults(t, expectedResult, result)
}

// executeWalShow invokes the HandleWalVerify() with fake storage filled with
// provided wal segments and other storage folder files
func executeWalVerify(
	walFilenames []string,
	storageFiles map[string]*bytes.Buffer,
	currentWalSegment postgres.WalSegmentDescription,
) (map[postgres.WalVerifyCheckType]postgres.WalVerifyCheckResult, int) {
	rootFolder := setupTestStorageFolder()
	walFolder := rootFolder.GetSubFolder(utility.WalPath)
	for name, content := range storageFiles {
		_ = rootFolder.PutObject(name, content)
	}
	putWalSegments(walFilenames, walFolder)

	mockOutputWriter := &MockWalVerifyOutputWriter{}
	checkTypes := []postgres.WalVerifyCheckType{
		postgres.WalVerifyTimelineCheck, postgres.WalVerifyIntegrityCheck}

	postgres.HandleWalVerify(checkTypes, rootFolder, currentWalSegment, mockOutputWriter)

	return mockOutputWriter.lastResult, mockOutputWriter.writeCallsCount
}

func compareResults(
	t *testing.T,
	expected map[postgres.WalVerifyCheckType]postgres.WalVerifyCheckResult,
	returned map[postgres.WalVerifyCheckType]postgres.WalVerifyCheckResult) {

	assert.Equal(t, len(expected), len(returned))

	for checkType, checkResult := range returned {
		assert.Contains(t, expected, checkType)
		assert.Equal(t, expected[checkType].Status, checkResult.Status,
			"Result status doesn't match the expected status")

		assert.True(t, reflect.DeepEqual(expected[checkType].Details, checkResult.Details),
			"Result details don't match the expected values")
	}
}

func newMockExtendedMetadataDto(isPermanent bool) postgres.ExtendedMetadataDto {
	// currently we do not need any fields except the isPermanent
	return postgres.ExtendedMetadataDto{
		StartTime:        time.Now(),
		FinishTime:       time.Now().Add(time.Second),
		DatetimeFormat:   postgres.MetadataDatetimeFormat,
		Hostname:         "test_host",
		DataDir:          "",
		PgVersion:        10000,
		StartLsn:         0,
		FinishLsn:        0,
		IsPermanent:      isPermanent,
		SystemIdentifier: nil,
		UncompressedSize: 0,
		CompressedSize:   0,
		UserData:         nil,
	}
}
