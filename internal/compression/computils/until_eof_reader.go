package computils

import "io"

type UntilEOFReader struct {
	underlying io.Reader
	isEOF      bool
}

func NewUntilEOFReader(underlying io.Reader) *UntilEOFReader {
	return &UntilEOFReader{underlying, false}
}

func (reader *UntilEOFReader) Read(p []byte) (n int, err error) {
	if reader.isEOF {
		return 0, io.EOF
	}
	n, err = reader.underlying.Read(p)
	if err == io.EOF {
		reader.isEOF = true
	}
	return
}
