package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/stretchr/testify/assert"
)

func TestFSFolder(t *testing.T) {
	tmpDir := setupTmpDir(t)

	defer os.RemoveAll(tmpDir)
	var storageFolder storage.Folder

	storageFolder, err := ConfigureFolder(tmpDir, nil)

	assert.NoError(t, err)

	storage.RunFolderTest(storageFolder, t)
}

func setupTmpDir(t *testing.T) string {
	cwd, err := filepath.Abs("./")
	if err != nil {
		t.Log(err)
	}
	// Create temp directory.
	tmpDir, err := os.MkdirTemp(cwd, "data")
	if err != nil {
		t.Log(err)
	}
	err = os.Chmod(tmpDir, 0755)
	if err != nil {
		t.Log(err)
	}
	return tmpDir
}
