package limiters_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/apecloud/dataprotection-wal-g/internal/limiters"
	"github.com/apecloud/dataprotection-wal-g/utility"
)

type fakeCloser struct {
	r io.Reader
}

func (r *fakeCloser) Read(buf []byte) (int, error) {
	n, err := r.r.Read(buf)
	return n, err
}

func (r *fakeCloser) Close() error {
	return nil
}

func TestLimiter(t *testing.T) {
	limiters.DiskLimiter = rate.NewLimiter(rate.Limit(10000), int(1024))
	limiters.NetworkLimiter = rate.NewLimiter(rate.Limit(10000), int(1024))
	defer func() {
		limiters.DiskLimiter = nil
		limiters.NetworkLimiter = nil
	}()
	buffer := bytes.NewReader(make([]byte, 2000))
	r := &fakeCloser{buffer}
	start := utility.TimeNowCrossPlatformLocal()

	reader := limiters.NewDiskLimitReader(limiters.NewNetworkLimitReader(r))
	_, err := io.ReadAll(reader)
	assert.NoError(t, err)
	end := utility.TimeNowCrossPlatformLocal()

	if end.Sub(start) < time.Millisecond*80 {
		t.Errorf("Rate limiter did not work")
	}
}
