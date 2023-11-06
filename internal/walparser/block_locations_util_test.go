package walparser_test

import (
	"testing"

	"github.com/apecloud/dataprotection-wal-g/internal/walparser"
	"github.com/apecloud/dataprotection-wal-g/testtools"
	"github.com/stretchr/testify/assert"
)

func TestExtractBlockLocations(t *testing.T) {
	record, _ := testtools.GetXLogRecordData()
	expectedLocations := []walparser.BlockLocation{record.Blocks[0].Header.BlockLocation}
	actualLocations := walparser.ExtractBlockLocations([]walparser.XLogRecord{record})
	assert.Equal(t, expectedLocations, actualLocations)
}
