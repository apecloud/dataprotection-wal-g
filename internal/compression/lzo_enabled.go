//go:build lzo && !windows
// +build lzo,!windows

package compression

import "github.com/apecloud/dataprotection-wal-g/internal/compression/lzo"

func init() {
	Decompressors = append(Decompressors, lzo.Decompressor{})
}
