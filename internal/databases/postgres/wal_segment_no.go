package postgres

import (
	"fmt"
)

type WalSegmentNo uint64

func newWalSegmentNo(lsn LSN) WalSegmentNo {
	return WalSegmentNo(getSegmentNoFromLsn(lsn))
}

func getSegmentNoFromLsn(lsn LSN) uint64 {
	return uint64(lsn) / WalSegmentSize
}

func newWalSegmentNoFromFilename(filename string) (WalSegmentNo, error) {
	_, no, err := ParseWALFilename(filename)
	return WalSegmentNo(no), err
}

func (walSegmentNo WalSegmentNo) next() WalSegmentNo {
	return walSegmentNo.add(1)
}

func (walSegmentNo WalSegmentNo) previous() WalSegmentNo {
	return walSegmentNo.sub(1)
}

func (walSegmentNo WalSegmentNo) add(n uint64) WalSegmentNo {
	return WalSegmentNo(uint64(walSegmentNo) + n)
}

func (walSegmentNo WalSegmentNo) sub(n uint64) WalSegmentNo {
	return WalSegmentNo(uint64(walSegmentNo) - n)
}

func (walSegmentNo WalSegmentNo) firstLsn() LSN {
	return LSN(uint64(walSegmentNo) * WalSegmentSize)
}

func (walSegmentNo WalSegmentNo) getFilename(timeline uint32) string {
	return fmt.Sprintf(walFileFormat,
		timeline, uint64(walSegmentNo)/xLogSegmentsPerXLogID,
		uint64(walSegmentNo)%xLogSegmentsPerXLogID)
}
