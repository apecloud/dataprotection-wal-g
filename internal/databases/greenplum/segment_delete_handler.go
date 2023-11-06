package greenplum

import (
	"fmt"
	"path"
	"strings"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

type SegDeleteType int

const (
	SegDeleteBefore SegDeleteType = iota
	SegDeleteTarget
)

type SegDeleteHandler interface {
	Delete(backup SegBackup) error
}

type SegDeleteBeforeHandler struct {
	*postgres.DeleteHandler
	contentID int
	args      DeleteArgs
}

func NewSegDeleteHandler(rootFolder storage.Folder, contentID int, args DeleteArgs, delType SegDeleteType,
) (SegDeleteHandler, error) {
	segFolder := rootFolder.GetSubFolder(FormatSegmentStoragePrefix(contentID))
	permanentBackups, permanentWals := postgres.GetPermanentBackupsAndWals(segFolder)

	segDeleteHandler, err := postgres.NewDeleteHandler(segFolder, permanentBackups, permanentWals, false)
	if err != nil {
		return nil, err
	}

	switch delType {
	case SegDeleteBefore:
		return &SegDeleteBeforeHandler{
			DeleteHandler: segDeleteHandler,
			contentID:     contentID,
			args:          args,
		}, nil

	case SegDeleteTarget:
		return &SegDeleteTargetHandler{
			DeleteHandler: segDeleteHandler,
			contentID:     contentID,
			args:          args,
		}, nil

	default:
		return nil, fmt.Errorf("unknown segment delete handler type")
	}
}

func (h SegDeleteBeforeHandler) Delete(segBackup SegBackup) error {
	segTarget, err := h.FindTargetByName(segBackup.Name)
	if err != nil {
		return err
	}

	tracelog.InfoLogger.Printf("Running delete before target %s on segment %d\n",
		segTarget.GetBackupName(), h.contentID)

	filterFunc := func(object storage.Object) bool { return true }
	folderFilter := func(folderPath string) bool {
		aoSegFolderPrefix := path.Join(utility.BaseBackupPath, AoStoragePath)
		return !strings.HasPrefix(folderPath, aoSegFolderPrefix)
	}
	err = h.DeleteBeforeTargetWhere(segTarget, h.args.Confirmed, filterFunc, folderFilter)
	if err != nil {
		return err
	}

	return cleanupAOSegments(segTarget, h.Folder, h.args.Confirmed)
}

type SegDeleteTargetHandler struct {
	*postgres.DeleteHandler
	contentID int
	args      DeleteArgs
}

func (h SegDeleteTargetHandler) Delete(segBackup SegBackup) error {
	segTarget, err := h.FindTargetByName(segBackup.Name)
	if err != nil {
		return err
	}

	tracelog.InfoLogger.Printf("Running delete target %s on segment %d\n",
		segTarget.GetBackupName(), h.contentID)

	folderFilter := func(folderPath string) bool {
		return !strings.HasPrefix(folderPath, AoStoragePath)
	}
	err = h.DeleteTarget(segTarget, h.args.Confirmed, h.args.FindFull, folderFilter)
	if err != nil {
		return err
	}

	return cleanupAOSegments(segTarget, h.Folder, h.args.Confirmed)
}

// TODO: unit tests
func cleanupAOSegments(target internal.BackupObject, segFolder storage.Folder, confirmed bool) error {
	aoSegFolder := segFolder.GetSubFolder(utility.BaseBackupPath).GetSubFolder(AoStoragePath)
	aoSegmentsToRetain, err := LoadStorageAoFiles(segFolder.GetSubFolder(utility.BaseBackupPath))
	if err != nil {
		return err
	}

	for segPath := range aoSegmentsToRetain {
		tracelog.DebugLogger.Printf("%s is still used, retaining...\n", segPath)
	}

	tracelog.InfoLogger.Printf("Cleaning up the AO segment objects")
	aoSegmentsToDelete, err := findAoSegmentsToDelete(target, aoSegmentsToRetain, aoSegFolder)
	if err != nil {
		return err
	}

	if !confirmed {
		return nil
	}

	return aoSegFolder.DeleteObjects(aoSegmentsToDelete)
}

// TODO: unit tests
func findAoSegmentsToDelete(target internal.BackupObject,
	aoSegmentsToRetain map[string]struct{}, aoSegFolder storage.Folder) ([]string, error) {
	aoObjects, _, err := aoSegFolder.ListFolder()
	if err != nil {
		return nil, err
	}

	aoSegmentsToDelete := make([]string, 0)
	for _, obj := range aoObjects {
		if !strings.HasSuffix(obj.GetName(), AoSegSuffix) && obj.GetLastModified().After(target.GetLastModified()) {
			tracelog.DebugLogger.Println(
				"\tis not an AO segment file, will not delete (modify time is too recent): " + obj.GetName())
			continue
		}

		if _, ok := aoSegmentsToRetain[obj.GetName()]; ok {
			// this AO segment file is still referenced by some backup, skip it
			tracelog.DebugLogger.Println("\tis still referenced by some backups, will not delete: " + obj.GetName())
			continue
		}

		tracelog.InfoLogger.Println("\twill be deleted: " + obj.GetName())

		aoSegmentsToDelete = append(aoSegmentsToDelete, obj.GetName())
	}

	return aoSegmentsToDelete, nil
}
