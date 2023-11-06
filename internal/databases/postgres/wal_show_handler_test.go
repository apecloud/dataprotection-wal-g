package postgres_test

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"

	"github.com/apecloud/dataprotection-wal-g/internal/compression"
	"github.com/apecloud/dataprotection-wal-g/internal/compression/lz4"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/memory"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/stretchr/testify/assert"
)

// MockWalShowOutputWriter is used to capture wal-show command output
type MockWalShowOutputWriter struct {
	timelineInfos []*postgres.TimelineInfo
}

func (writer *MockWalShowOutputWriter) Write(timelineInfos []*postgres.TimelineInfo) error {
	// append timeline infos in case future implementations will call the Write() multiple times
	writer.timelineInfos = append(writer.timelineInfos, timelineInfos...)
	return nil
}

// TestTimelineSetup holds test setup information about single timeline
type TestTimelineSetup struct {
	existSegments       []string
	missingSegments     []string
	id                  uint32
	parentId            uint32
	switchPointLsn      postgres.LSN
	historyFileContents string
}

// GetWalFilenames returns slice of existing wal segments filenames
func (timelineSetup *TestTimelineSetup) GetWalFilenames() []string {
	walFileSuffix := "." + lz4.FileExtension
	filenamesWithExtension := make([]string, 0, len(timelineSetup.existSegments))
	for _, name := range timelineSetup.existSegments {
		filenamesWithExtension = append(filenamesWithExtension, name+walFileSuffix)
	}
	return filenamesWithExtension
}

// newTimelineHistoryFile returns .history file name and compressed contents
func newTimelineHistoryFile(contents string, timelineId uint32) (string, *bytes.Buffer, error) {
	compressor := compression.Compressors[lz4.AlgorithmName]
	var compressedData bytes.Buffer
	compressingWriter := compressor.NewWriter(&compressedData)
	_, err := utility.FastCopy(compressingWriter, strings.NewReader(contents))
	if err != nil {
		return "", nil, err
	}
	err = compressingWriter.Close()
	if err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("%08X.history."+lz4.FileExtension, timelineId), &compressedData, nil
}

// TestWalShow test series is used to test the HandleWalShow() functionality

func TestWalShow_NoSegmentsInStorage(t *testing.T) {
	timelineInfos := executeWalShow([]string{}, make(map[string]*bytes.Buffer))
	assert.Empty(t, timelineInfos)
}

func TestWalShow_NoMissingSegments(t *testing.T) {
	timelineSetup := &TestTimelineSetup{
		existSegments: []string{
			"000000010000000000000090",
			"000000010000000000000091",
			"000000010000000000000092",
			"000000010000000000000093",
		},
		missingSegments: make([]string, 0),
		id:              1,
	}
	testSingleTimeline(t, timelineSetup, make(map[string]*bytes.Buffer))
}

func TestWalShow_OneSegmentMissing(t *testing.T) {
	timelineSetup := &TestTimelineSetup{
		existSegments: []string{
			"000000010000000000000090",
			"000000010000000000000092",
			"000000010000000000000093",
			"000000010000000000000094",
		},
		missingSegments: []string{
			"000000010000000000000091",
		},
		id: 1,
	}
	testSingleTimeline(t, timelineSetup, make(map[string]*bytes.Buffer))
}

func TestWalShow_MultipleSegmentsMissing(t *testing.T) {
	timelineSetup := &TestTimelineSetup{
		existSegments: []string{
			"000000010000000000000090",
			"000000010000000000000092",
			"000000010000000000000093",
			"000000010000000000000095",
		},
		missingSegments: []string{
			"000000010000000000000091",
			"000000010000000000000094",
		},
		id: 1,
	}
	testSingleTimeline(t, timelineSetup, make(map[string]*bytes.Buffer))
}

func TestWalShow_SingleTimelineWithHistory(t *testing.T) {
	timelineSetup := &TestTimelineSetup{
		existSegments: []string{
			"000000020000000000000090",
			"000000020000000000000091",
			"000000020000000000000092",
			"000000020000000000000093",
		},
		missingSegments: make([]string, 0),
		id:              2,
		// parentId and switch point LSN match the .history file record
		parentId: 1,
		// 2420113408 is 0x90400000 (hex)
		switchPointLsn:      2420113408,
		historyFileContents: "1\t0/90400000\tbefore 2000-01-01 05:00:00+05\n\n",
	}

	fileName, contents, err := newTimelineHistoryFile(
		timelineSetup.historyFileContents, timelineSetup.id)
	assert.NoError(t, err)

	testSingleTimeline(t, timelineSetup, map[string]*bytes.Buffer{fileName: contents})
}

func TestWalShow_TwoTimelinesWithHistory(t *testing.T) {
	timelineSetups := []*TestTimelineSetup{
		{
			existSegments: []string{
				"00000001000000000000008F",
				"000000010000000000000090",
				"000000010000000000000091",
				"000000010000000000000092",
			},
			missingSegments: make([]string, 0),
			id:              1,
		},
		{
			existSegments: []string{
				"000000020000000000000090",
				"000000020000000000000091",
				"000000020000000000000092",
			},
			missingSegments: make([]string, 0),
			id:              2,
			// parentId and switch point LSN match the .history file record
			parentId: 1,
			// 2420113408 is 0x90400000 (hex)
			switchPointLsn:      2420113408,
			historyFileContents: "1\t0/90400000\tbefore 2000-01-01 05:00:00+05\n\n",
		},
	}

	fileName, contents, err := newTimelineHistoryFile(
		timelineSetups[1].historyFileContents, timelineSetups[1].id)
	assert.NoError(t, err)

	testMultipleTimelines(t, timelineSetups, map[string]*bytes.Buffer{
		fileName: contents,
	})
}

func TestWalShow_TwoTimelinesWithHistory_HighTLI(t *testing.T) {
	timelineSetups := []*TestTimelineSetup{
		{
			existSegments: []string{
				"00EEEEED000000000000008F",
				"00EEEEED0000000000000090",
				"00EEEEED0000000000000091",
				"00EEEEED0000000000000092",
			},
			missingSegments: make([]string, 0),
			id:              15658733,
		},
		{
			existSegments: []string{
				"00EEEEEE0000000000000090",
				"00EEEEEE0000000000000091",
				"00EEEEEE0000000000000092",
			},
			missingSegments: make([]string, 0),
			// 15658734 is 0xEEEEEE (hex)
			id: 15658734,
			// parentId and switch point LSN match the .history file record
			parentId: 15658733,
			// 2420113408 is 0x90400000 (hex)
			switchPointLsn:      2420113408,
			historyFileContents: "15658733\t0/90400000\tbefore 2000-01-01 05:00:00+05\n\n",
		},
	}

	fileName, contents, err := newTimelineHistoryFile(
		timelineSetups[1].historyFileContents, timelineSetups[1].id)
	assert.NoError(t, err)

	testMultipleTimelines(t, timelineSetups, map[string]*bytes.Buffer{
		fileName: contents,
	})
}

func TestWalShow_MultipleTimelines(t *testing.T) {
	timelineSetups := []*TestTimelineSetup{
		// first timeline
		{
			existSegments: []string{
				"000000010000000000000090",
				"000000010000000000000091",
				"000000010000000000000092",
				"000000010000000000000093",
			},
			id: 1,
		},
		// second timeline
		{
			existSegments: []string{
				"000000020000000000000091",
				"000000020000000000000092",
			},
			id: 2,
		},
	}
	testMultipleTimelines(t, timelineSetups, make(map[string]*bytes.Buffer))
}

// testSingleTimeline is used to test wal-show with only one timeline in WAL storage
func testSingleTimeline(t *testing.T, setup *TestTimelineSetup, walFolderFiles map[string]*bytes.Buffer) {
	timelines := executeWalShow(setup.GetWalFilenames(), walFolderFiles)
	assert.Len(t, timelines, 1)

	verifySingleTimeline(t, setup, timelines[0])
}

// testMultipleTimelines is used to test wal-show in case of multiple timelines in WAL storage
func testMultipleTimelines(t *testing.T, timelineSetups []*TestTimelineSetup, walFolderFiles map[string]*bytes.Buffer) {
	walFilenames := concatWalFilenames(timelineSetups)
	timelineInfos := executeWalShow(walFilenames, walFolderFiles)

	sort.Slice(timelineInfos, func(i, j int) bool {
		return timelineInfos[i].ID < timelineInfos[j].ID
	})
	sort.Slice(timelineSetups, func(i, j int) bool {
		return timelineSetups[i].id < timelineSetups[j].id
	})

	assert.Len(t, timelineInfos, len(timelineSetups))

	for idx, info := range timelineInfos {
		verifySingleTimeline(t, timelineSetups[idx], info)
	}
}

// verifySingleTimeline checks that setup values for timeline matches the output timeline info values
func verifySingleTimeline(t *testing.T, setup *TestTimelineSetup, timelineInfo *postgres.TimelineInfo) {
	// sort setup.existSegments to pick the correct start and end segment
	sort.Slice(setup.existSegments, func(i, j int) bool {
		return setup.existSegments[i] < setup.existSegments[j]
	})

	expectedStatus := postgres.TimelineOkStatus
	if len(setup.missingSegments) > 0 {
		expectedStatus = postgres.TimelineLostSegmentStatus
	}

	expectedTimelineInfo := postgres.TimelineInfo{
		ID:               setup.id,
		ParentID:         setup.parentId,
		SwitchPointLsn:   setup.switchPointLsn,
		StartSegment:     setup.existSegments[0],
		EndSegment:       setup.existSegments[len(setup.existSegments)-1],
		SegmentsCount:    len(setup.existSegments),
		MissingSegments:  setup.missingSegments,
		SegmentRangeSize: uint64(len(setup.existSegments) + len(setup.missingSegments)),
		Status:           expectedStatus,
	}

	// check that found missing segments matches with setup values
	assert.ElementsMatch(t, expectedTimelineInfo.MissingSegments, timelineInfo.MissingSegments)

	// avoid equality errors (we ignore missing segments order and we've checked that MissingSegments match before)
	expectedTimelineInfo.MissingSegments = timelineInfo.MissingSegments
	assert.Equal(t, expectedTimelineInfo, *timelineInfo)
}

// executeWalShow invokes the HandleWalShow() with fake storage filled with
// provided wal segments and other folder files
func executeWalShow(walFilenames []string, walFolderFiles map[string]*bytes.Buffer) []*postgres.TimelineInfo {
	rootFolder := setupTestStorageFolder()
	walFolder := rootFolder.GetSubFolder(utility.WalPath)
	putWalSegments(walFilenames, walFolder)

	for name, content := range walFolderFiles {
		_ = walFolder.PutObject(name, content)
	}

	mockOutputWriter := &MockWalShowOutputWriter{}
	postgres.HandleWalShow(rootFolder, false, mockOutputWriter)

	return mockOutputWriter.timelineInfos
}

func putWalSegments(walFilenames []string, walFolder storage.Folder) {
	for _, name := range walFilenames {
		// we don't use the WAL file contents so let it be it empty inside
		_ = walFolder.PutObject(name, new(bytes.Buffer))
	}
}

func setupTestStorageFolder() storage.Folder {
	memoryStorage := memory.NewStorage()
	return memory.NewFolder("in_memory/", memoryStorage)
}

func concatWalFilenames(timelineSetups []*TestTimelineSetup) []string {
	filenames := make([]string, 0)
	for _, timelineSetup := range timelineSetups {
		filenames = append(filenames, timelineSetup.GetWalFilenames()...)
	}
	return filenames
}
