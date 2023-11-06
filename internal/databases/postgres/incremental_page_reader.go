package postgres

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/RoaringBitmap/roaring"
	"github.com/apecloud/dataprotection-wal-g/internal/ioextensions"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

// IncrementFileHeader contains "wi" at the head which stands for "wal-g increment"
// format version "1", signature magic number
var IncrementFileHeader = []byte{'w', 'i', '1', SignatureMagicNumber}

// IncrementalPageReader constructs difference map during initialization and than re-read file
// Diff map may consist of 1Gb/PostgresBlockSize elements == 512Kb
type IncrementalPageReader struct {
	PagedFile ioextensions.ReadSeekCloser
	FileSize  int64
	Lsn       LSN
	Next      []byte
	Blocks    []uint32
}

func (pageReader *IncrementalPageReader) Read(p []byte) (n int, err error) {
	for {
		copied := copy(p, pageReader.Next)
		p = p[copied:]
		pageReader.Next = pageReader.Next[copied:]
		n += copied
		if len(p) == 0 {
			return n, nil
		}
		moreData, err := pageReader.DrainMoreData()
		if err != nil {
			return n, err
		}
		if !moreData {
			return n, io.EOF
		}
	}
}

func (pageReader *IncrementalPageReader) DrainMoreData() (succeed bool, err error) {
	if len(pageReader.Blocks) == 0 {
		return false, nil
	}
	err = pageReader.AdvanceFileReader()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (pageReader *IncrementalPageReader) AdvanceFileReader() error {
	pageBytes := make([]byte, DatabasePageSize)
	blockNo := pageReader.Blocks[0]
	pageReader.Blocks = pageReader.Blocks[1:]
	offset := int64(blockNo) * DatabasePageSize
	// TODO : possible race condition - page was deleted between blocks extraction and seek
	_, err := pageReader.PagedFile.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = io.ReadFull(pageReader.PagedFile, pageBytes)
	if err == nil {
		pageReader.Next = pageBytes
	}
	return err
}

// Close IncrementalPageReader
func (pageReader *IncrementalPageReader) Close() error {
	return pageReader.PagedFile.Close()
}

// TODO : unit tests
// TODO : "initialize" is rather meaningless name, maybe this func should be decomposed
func (pageReader *IncrementalPageReader) initialize(deltaBitmap *roaring.Bitmap) (size int64, err error) {
	var headerBuffer bytes.Buffer
	headerBuffer.Write(IncrementFileHeader)
	fileSize := pageReader.FileSize
	headerBuffer.Write(utility.ToBytes(uint64(fileSize)))
	pageReader.Blocks = make([]uint32, 0, fileSize/DatabasePageSize)

	if deltaBitmap == nil {
		err := pageReader.FullScanInitialize()
		if err != nil {
			return 0, err
		}
	} else {
		pageReader.DeltaBitmapInitialize(deltaBitmap)
	}

	pageReader.WriteDiffMapToHeader(&headerBuffer)
	pageReader.Next = headerBuffer.Bytes()
	pageDataSize := int64(len(pageReader.Blocks)) * DatabasePageSize
	size = int64(headerBuffer.Len()) + pageDataSize
	return
}

func (pageReader *IncrementalPageReader) DeltaBitmapInitialize(deltaBitmap *roaring.Bitmap) {
	it := deltaBitmap.Iterator()
	for it.HasNext() { // TODO : do something with file truncation during reading
		blockNo := it.Next()
		if pageReader.FileSize >= int64(blockNo+1)*DatabasePageSize { // whole block fits into file
			pageReader.Blocks = append(pageReader.Blocks, blockNo)
		} else {
			break
		}
	}
}

func (pageReader *IncrementalPageReader) FullScanInitialize() error {
	pageBytes := make([]byte, DatabasePageSize)
	for currentBlockNumber := uint32(0); ; currentBlockNumber++ {
		_, err := io.ReadFull(pageReader.PagedFile, pageBytes)

		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}

		valid := pageReader.SelectNewValidPage(pageBytes, currentBlockNumber) // TODO : torn page possibility
		if !valid {
			return newInvalidBlockError(currentBlockNumber)
		}
	}
}

// WriteDiffMapToHeader is currently used only with buffers, so we don't handle any writing errors
func (pageReader *IncrementalPageReader) WriteDiffMapToHeader(headerWriter io.Writer) {
	diffBlockCount := len(pageReader.Blocks)
	_, _ = headerWriter.Write(utility.ToBytes(uint32(diffBlockCount)))

	for _, blockNo := range pageReader.Blocks {
		_ = binary.Write(headerWriter, binary.LittleEndian, blockNo)
	}
}

// SelectNewValidPage checks whether page is valid and if it so, then blockNo is appended to Blocks list
func (pageReader *IncrementalPageReader) SelectNewValidPage(pageBytes []byte, blockNo uint32) (valid bool) {
	pageHeader, _ := parsePostgresPageHeader(bytes.NewReader(pageBytes))
	valid = pageHeader.isValid()

	isNew := false

	if !valid {
		if pageHeader.isNew() { // vacuumed page
			isNew = true
			valid = true
		} else {
			tracelog.DebugLogger.Println("Invalid page ", blockNo, " page header ", pageHeader)
			return false
		}
	}

	if isNew || (pageHeader.lsn() >= pageReader.Lsn) {
		pageReader.Blocks = append(pageReader.Blocks, blockNo)
	}
	return
}
