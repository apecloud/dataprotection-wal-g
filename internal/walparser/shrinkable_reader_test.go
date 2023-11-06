package walparser

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShrinkableReader_NotEnoughDataToShrinkError(t *testing.T) {
	reader := ShrinkableReader{nil, 0}
	err := reader.Shrink(5)
	if _, ok := err.(NotEnoughDataToShrinkError); !ok {
		t.Fatalf("expected shrinking error, but got: %v", err)
	}
}

func TestShrinkableReader_Read(t *testing.T) {
	reader := bytes.NewReader([]byte{1, 2, 3})
	shrinkableReader := ShrinkableReader{reader, 2}
	buf := make([]byte, 10)
	readCount, err := shrinkableReader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, readCount, 2)
	assert.Equal(t, buf[:2], []byte{1, 2})
}

func TestShrinkableReader_Shrink(t *testing.T) {
	reader := bytes.NewReader([]byte{1, 2, 3, 4, 5})
	shrinkableReader := ShrinkableReader{reader, 4}
	err := shrinkableReader.Shrink(1)
	assert.NoError(t, err)
	buf := make([]byte, 4)
	readCount, err := shrinkableReader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, readCount, 3)
}

func TestShrinkableReader_EOF(t *testing.T) {
	reader := bytes.NewReader([]byte{1, 2, 3, 4, 5})
	shrinkableReader := ShrinkableReader{reader, 5}
	err := shrinkableReader.Shrink(5)
	assert.NoError(t, err)
	AssertReaderIsEmpty(t, &shrinkableReader)
}
