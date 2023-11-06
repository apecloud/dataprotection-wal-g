package postgres

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/apecloud/dataprotection-wal-g/internal"

	"github.com/RoaringBitmap/roaring"
	"github.com/apecloud/dataprotection-wal-g/internal/walparser"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

const (
	RelFileSizeBound               = 1 << 30
	BlocksInRelFile                = int(RelFileSizeBound / DatabasePageSize)
	DefaultSpcNode   walparser.Oid = 1663
)

type NoBitmapFoundError struct {
	error
}

func newNoBitmapFoundError() NoBitmapFoundError {
	return NoBitmapFoundError{errors.New("GetDeltaBitmapFor: no bitmap found")}
}

func (err NoBitmapFoundError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

type UnknownTableSpaceError struct {
	error
}

func newUnknownTableSpaceError() UnknownTableSpaceError {
	return UnknownTableSpaceError{errors.New("GetRelFileNodeFrom: unknown tablespace")}
}

func (err UnknownTableSpaceError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

type PagedFileDeltaMap map[walparser.RelFileNode]*roaring.Bitmap

func NewPagedFileDeltaMap() PagedFileDeltaMap {
	deltaMap := make(map[walparser.RelFileNode]*roaring.Bitmap)
	pagedFileDeltaMap := PagedFileDeltaMap(deltaMap)
	return pagedFileDeltaMap
}

func (deltaMap *PagedFileDeltaMap) AddLocationToDelta(location walparser.BlockLocation) {
	_, contains := (*deltaMap)[location.RelationFileNode]
	if !contains {
		(*deltaMap)[location.RelationFileNode] = roaring.BitmapOf(location.BlockNo)
	} else {
		bitmap := (*deltaMap)[location.RelationFileNode]
		bitmap.Add(location.BlockNo)
	}
}

func (deltaMap *PagedFileDeltaMap) AddLocationsToDelta(locations []walparser.BlockLocation) {
	for _, location := range locations {
		deltaMap.AddLocationToDelta(location)
	}
}

// TODO : unit test no bitmap found
func (deltaMap *PagedFileDeltaMap) GetDeltaBitmapFor(filePath string) (*roaring.Bitmap, error) {
	relFileNode, err := GetRelFileNodeFrom(filePath)
	if err != nil {
		return nil, err
	}
	_, ok := (*deltaMap)[*relFileNode]
	if !ok {
		return nil, newNoBitmapFoundError()
	}
	bitmap := (*deltaMap)[*relFileNode].Clone()
	relFileID, err := GetRelFileIDFrom(filePath)
	if err != nil {
		return nil, err
	}
	return SelectRelFileBlocks(bitmap, relFileID), nil
}

func SelectRelFileBlocks(bitmap *roaring.Bitmap, relFileID int) *roaring.Bitmap {
	relFileBitmap := roaring.New()
	relFileBitmap.AddRange(uint64(relFileID*BlocksInRelFile), uint64((relFileID+1)*BlocksInRelFile))
	bitmap.And(relFileBitmap)
	shiftedBitmap := roaring.New()
	it := bitmap.Iterator()
	for it.HasNext() {
		shiftedBitmap.Add(it.Next() - uint32(relFileID*BlocksInRelFile))
	}
	return shiftedBitmap
}

func GetRelFileIDFrom(filePath string) (int, error) {
	filename := path.Base(filePath)
	match := pagedFilenameRegexp.FindStringSubmatch(filename)
	if match[2] == "" {
		return 0, nil
	}
	return strconv.Atoi(match[2][1:])
}

func GetRelFileNodeFrom(filePath string) (*walparser.RelFileNode, error) {
	folderPath, name := path.Split(filePath)
	folderPathParts := strings.Split(strings.TrimSuffix(folderPath, "/"), string(os.PathSeparator))
	match := pagedFilenameRegexp.FindStringSubmatch(name)
	if match == nil {
		return nil, errors.New("GetRelFileNodeFrom: can't parse path: " + filePath)
	}
	relNode, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, errors.Wrapf(err, "GetRelFileNodeFrom: can't get relNode from: '%s'", filePath)
	}
	dbNode, err := strconv.Atoi(folderPathParts[len(folderPathParts)-1])
	if err != nil {
		return nil, errors.Wrapf(err, "GetRelFileNodeFrom: can't get dbNode from: '%s'", filePath)
	}
	if strings.Contains(filePath, DefaultTablespace) { // base
		return &walparser.RelFileNode{SpcNode: DefaultSpcNode,
			DBNode:  walparser.Oid(dbNode),
			RelNode: walparser.Oid(relNode)}, nil
	} else if strings.Contains(filePath, NonDefaultTablespace) { // pg_tblspc
		spcNode, err := strconv.Atoi(folderPathParts[len(folderPathParts)-3])
		if err != nil {
			return nil, err
		}
		return &walparser.RelFileNode{SpcNode: walparser.Oid(spcNode),
			DBNode:  walparser.Oid(dbNode),
			RelNode: walparser.Oid(relNode)}, nil
	} else {
		return nil, newUnknownTableSpaceError()
	}
}

func (deltaMap *PagedFileDeltaMap) getLocationsFromDeltas(reader internal.StorageFolderReader,
	timeline uint32,
	first,
	last DeltaNo) error {
	for deltaNo := first; deltaNo < last; deltaNo = deltaNo.next() {
		filename := deltaNo.getFilename(timeline)
		deltaFile, err := getDeltaFile(reader, filename)
		if err != nil {
			return err
		}
		tracelog.InfoLogger.Printf("Successfully downloaded delta file %s\n", filename)
		deltaMap.AddLocationsToDelta(deltaFile.Locations)
	}
	return nil
}

func (deltaMap *PagedFileDeltaMap) getLocationsFromWals(reader internal.StorageFolderReader,
	timeline uint32,
	first,
	last WalSegmentNo,
	walParser *walparser.WalParser) error {
	for walSegmentNo := first; walSegmentNo < last; walSegmentNo = walSegmentNo.next() {
		filename := walSegmentNo.getFilename(timeline)
		err := deltaMap.getLocationsFromWal(reader, filename, walParser)
		if err != nil {
			return err
		}
		tracelog.InfoLogger.Printf("Successfully downloaded wal file %s\n", filename)
	}
	return nil
}

func (deltaMap *PagedFileDeltaMap) getLocationsFromWal(
	folderReader internal.StorageFolderReader, filename string, walParser *walparser.WalParser) error {
	reader, err := internal.DownloadAndDecompressStorageFile(folderReader, filename)
	if err != nil {
		return errors.Wrapf(err, "Error during wal segment'%s' downloading.", filename)
	}
	locations, err := walparser.ExtractLocationsFromWalFile(walParser, reader)
	if err != nil {
		return errors.Wrapf(err, "Error during extracting locations from wal segment: '%s'", filename)
	}
	err = reader.Close()
	if err != nil {
		return errors.Wrapf(err, "Error during reading wal segment '%s'", filename)
	}
	deltaMap.AddLocationsToDelta(locations)
	return nil
}

func getDeltaFile(folderReader internal.StorageFolderReader, filename string) (*DeltaFile, error) {
	reader, err := internal.DownloadAndDecompressStorageFile(folderReader, filename)
	if err != nil {
		return nil, errors.Wrapf(err, "Error during delta file '%s' downloading.", filename)
	}
	deltaFile, err := LoadDeltaFile(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "Error during extracting locations from delta file: '%s'", filename)
	}
	err = reader.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "Error during reading delta file '%s'", filename)
	}
	return deltaFile, nil
}
