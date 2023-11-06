package zstd

import (
	"io"

	"github.com/apecloud/dataprotection-wal-g/internal/compression/computils"
	"github.com/klauspost/compress/zstd"
)

type Decompressor struct{}

func (decompressor Decompressor) Decompress(src io.Reader) (io.ReadCloser, error) {
	zstdReader, err := zstd.NewReader(computils.NewUntilEOFReader(src))
	if err != nil {
		return nil, err
	}
	return zstdReader.IOReadCloser(), nil
}

func (decompressor Decompressor) FileExtension() string {
	return FileExtension
}
