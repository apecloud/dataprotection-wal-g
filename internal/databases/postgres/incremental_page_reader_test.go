package postgres_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"

	"github.com/RoaringBitmap/roaring"
	"github.com/apecloud/dataprotection-wal-g/internal/ioextensions"
	"github.com/apecloud/dataprotection-wal-g/testtools"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/stretchr/testify/assert"
)

func TestDeltaBitmapInitialize(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{
		FileSize: postgres.DatabasePageSize * 5,
		Blocks:   make([]uint32, 0),
	}
	deltaBitmap := roaring.BitmapOf(0, 2, 3, 12, 14)
	pageReader.DeltaBitmapInitialize(deltaBitmap)
	assert.Equal(t, pageReader.Blocks, []uint32{0, 2, 3})
}

func TestSelectNewValidPage_ZeroPage(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{
		Blocks: make([]uint32, 0),
	}
	pageData := make([]byte, postgres.DatabasePageSize)
	var blockNo uint32 = 10
	valid := pageReader.SelectNewValidPage(pageData, blockNo)
	assert.True(t, valid)
	assert.Equal(t, []uint32{blockNo}, pageReader.Blocks)
}

func TestSelectNewValidPage_InvalidPage(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{
		Blocks: make([]uint32, 0),
	}
	pageData := make([]byte, postgres.DatabasePageSize)
	for i := byte(0); i < 24; i++ {
		pageData[i] = i
	}
	pageData[2134] = 100
	var blockNo uint32 = 10
	valid := pageReader.SelectNewValidPage(pageData, blockNo)
	assert.False(t, valid)
	assert.Equal(t, []uint32{}, pageReader.Blocks)
}

func TestSelectNewValidPage_ValidPageLowLsn(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{
		Blocks: make([]uint32, 0),
	}
	var blockNo uint32 = 10
	pageFile, err := os.Open(pagedFileName)
	assert.NoError(t, err)
	defer utility.LoggedClose(pageFile, "")
	pageData := make([]byte, postgres.DatabasePageSize)
	_, err = io.ReadFull(pageFile, pageData)
	assert.NoError(t, err)
	assert.NoError(t, err)
	valid := pageReader.SelectNewValidPage(pageData, blockNo)
	assert.True(t, valid)
	assert.Equal(t, []uint32{blockNo}, pageReader.Blocks)
}

func TestSelectNewValidPage_ValidPageHighLsn(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{
		Blocks: make([]uint32, 0),
		Lsn:    postgres.LSN(1) << 62,
	}
	var blockNo uint32 = 10
	pageFile, err := os.Open(pagedFileName)
	assert.NoError(t, err)
	defer utility.LoggedClose(pageFile, "")
	pageData := make([]byte, postgres.DatabasePageSize)
	_, err = io.ReadFull(pageFile, pageData)
	assert.NoError(t, err)
	assert.NoError(t, err)
	valid := pageReader.SelectNewValidPage(pageData, blockNo)
	assert.True(t, valid)
	assert.Equal(t, []uint32{}, pageReader.Blocks)
}

func TestWriteDiffMapToHeader(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{
		Blocks: []uint32{1, 2, 33},
	}
	var header bytes.Buffer
	pageReader.WriteDiffMapToHeader(&header)
	var diffBlockCount uint32
	err := binary.Read(&header, binary.LittleEndian, &diffBlockCount)
	assert.NoError(t, err)
	actualBlocks := make([]uint32, 0)
	for i := 0; i < int(diffBlockCount); i++ {
		var blockNo uint32
		err := binary.Read(&header, binary.LittleEndian, &blockNo)
		assert.NoError(t, err)
		actualBlocks = append(actualBlocks, blockNo)
	}
	testtools.AssertReaderIsEmpty(t, &header)
	assert.Equal(t, pageReader.Blocks, actualBlocks)
}

func TestFullScanInitialize(t *testing.T) {
	pageFile, err := os.Open(pagedFileName)
	defer utility.LoggedClose(pageFile, "")
	assert.NoError(t, err)
	pageReader := postgres.IncrementalPageReader{
		PagedFile: pageFile,
		Blocks:    make([]uint32, 0),
		Lsn:       sampleLSN,
	}
	err = pageReader.FullScanInitialize()
	assert.NoError(t, err)
	assert.Equal(t, []uint32{3, 4, 5, 6, 7}, pageReader.Blocks)
}

func makePageDataReader() ioextensions.ReadSeekCloser {
	pageCount := int64(8)
	pageData := make([]byte, pageCount*postgres.DatabasePageSize)
	for i := int64(0); i < pageCount; i++ {
		for j := i * postgres.DatabasePageSize; j < (i+1)*postgres.DatabasePageSize; j++ {
			pageData[j] = byte(i)
		}
	}
	pageDataReader := bytes.NewReader(pageData)
	return &ioextensions.ReadSeekCloserImpl{Reader: pageDataReader, Seeker: pageDataReader, Closer: &testtools.NopCloser{}}
}

func TestRead(t *testing.T) {
	blocks := []uint32{1, 2, 4}
	header := []byte{12, 13, 14}
	expectedRead := make([]byte, 3+3*postgres.DatabasePageSize)
	copy(expectedRead, header)
	for id, i := range blocks {
		for j := 3 + int64(id)*postgres.DatabasePageSize; j < 3+(int64(id)+1)*postgres.DatabasePageSize; j++ {
			expectedRead[j] = byte(i)
		}
	}

	pageReader := postgres.IncrementalPageReader{
		PagedFile: makePageDataReader(),
		Blocks:    blocks,
		Next:      header,
	}

	actualRead := make([]byte, 3+3*postgres.DatabasePageSize)
	_, err := io.ReadFull(&pageReader, actualRead)
	assert.NoError(t, err)
	assert.Equal(t, expectedRead, actualRead)
	testtools.AssertReaderIsEmpty(t, &pageReader)
}

func TestAdvanceFileReader(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{
		PagedFile: makePageDataReader(),
		Blocks:    []uint32{5, 9},
	}
	err := pageReader.AdvanceFileReader()
	assert.NoError(t, err)
	assert.Equal(t, []uint32{9}, pageReader.Blocks)
	expectedNext := make([]byte, postgres.DatabasePageSize)
	for i := int64(0); i < postgres.DatabasePageSize; i++ {
		expectedNext[i] = 5
	}
	assert.Equal(t, expectedNext, pageReader.Next)
}

func TestDrainMoreData_NoBlocks(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{}
	succeed, err := pageReader.DrainMoreData()
	assert.NoError(t, err)
	assert.False(t, succeed)
}

func TestDrainMoreData_HasBlocks(t *testing.T) {
	pageReader := postgres.IncrementalPageReader{
		PagedFile: makePageDataReader(),
		Blocks:    []uint32{3, 6},
	}
	succeed, err := pageReader.DrainMoreData()
	assert.NoError(t, err)
	assert.True(t, succeed)
	assert.Equal(t, []uint32{6}, pageReader.Blocks)
	expectedNext := make([]byte, postgres.DatabasePageSize)
	for i := int64(0); i < postgres.DatabasePageSize; i++ {
		expectedNext[i] = 3
	}
	assert.Equal(t, expectedNext, pageReader.Next)
}
