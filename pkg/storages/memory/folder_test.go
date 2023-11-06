package memory

import (
	"testing"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
)

func TestS3Folder(t *testing.T) {
	storage.RunFolderTest(NewFolder("in_memory/", NewStorage()), t)
}
