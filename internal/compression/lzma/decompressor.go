package lzma

import (
	"io"

	"github.com/apecloud/dataprotection-wal-g/internal/compression/computils"
	"github.com/ulikunitz/xz/lzma"
)

type Decompressor struct{}

func (decompressor Decompressor) Decompress(src io.Reader) (io.ReadCloser, error) {
	lzReader, err := lzma.NewReader(computils.NewUntilEOFReader(src))
	if err != nil {
		return nil, err
	}
	return io.NopCloser(lzReader), nil
}

func (decompressor Decompressor) FileExtension() string {
	return FileExtension
}
