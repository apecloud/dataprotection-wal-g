package postgres_test

import (
	"bytes"
	"testing"

	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"

	"github.com/apecloud/dataprotection-wal-g/internal/walparser"
	"github.com/stretchr/testify/assert"
)

func TestSaveLoadDeltaFile(t *testing.T) {
	deltaFile := &postgres.DeltaFile{
		Locations: []walparser.BlockLocation{
			*walparser.NewBlockLocation(1, 2, 3, 4),
			*walparser.NewBlockLocation(5, 6, 7, 8),
		},
		WalParser: walparser.NewWalParser(),
	}

	var deltaFileData bytes.Buffer
	err := deltaFile.Save(&deltaFileData)
	assert.NoError(t, err)

	loadedDeltaFile, err := postgres.LoadDeltaFile(&deltaFileData)
	assert.NoError(t, err)

	assert.Equal(t, deltaFile, loadedDeltaFile)
}
