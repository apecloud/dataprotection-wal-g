package postgres

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

type NonEmptyDBDataDirectoryError struct {
	error
}

func NewNonEmptyDBDataDirectoryError(dbDataDirectory string) NonEmptyDBDataDirectoryError {
	return NonEmptyDBDataDirectoryError{errors.Errorf("Directory %v for delta base must be empty", dbDataDirectory)}
}

func (err NonEmptyDBDataDirectoryError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

type PgControlNotFoundError struct {
	error
}

func newPgControlNotFoundError() PgControlNotFoundError {
	return PgControlNotFoundError{errors.Errorf("Expect pg_control archive, but not found")}
}

func (err PgControlNotFoundError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

func readRestoreSpec(path string, spec *TablespaceSpec) (err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read file: %v", err)
	}
	err = json.Unmarshal(data, spec)
	if err != nil {
		return fmt.Errorf("unable to unmarshal json: %v\n Full json data:\n %s", err, data)
	}

	return nil
}

// If specified - choose specified, else choose from latest sentinelDto
func chooseTablespaceSpecification(sentinelDtoSpec, spec *TablespaceSpec) *TablespaceSpec {
	// spec is preferred over sentinelDtoSpec.TablespaceSpec if it is non-nil
	if spec != nil {
		return spec
	} else if sentinelDtoSpec == nil {
		return &TablespaceSpec{}
	}
	return sentinelDtoSpec
}

// TODO : unit tests
// deltaFetchRecursion function composes Backup object and recursively searches for necessary base backup
func deltaFetchRecursionOld(backup Backup, folder storage.Folder, dbDataDirectory string,
	tablespaceSpec *TablespaceSpec, filesToUnwrap map[string]bool, extractProv ExtractProvider) error {
	sentinelDto, filesMetaDto, err := backup.GetSentinelAndFilesMetadata()
	if err != nil {
		return err
	}
	tablespaceSpec = chooseTablespaceSpecification(sentinelDto.TablespaceSpec, tablespaceSpec)
	sentinelDto.TablespaceSpec = tablespaceSpec

	if sentinelDto.IsIncremental() {
		tracelog.InfoLogger.Printf("Delta from %v at LSN %s \n", *(sentinelDto.IncrementFrom),
			*(sentinelDto.IncrementFromLSN))
		baseFilesToUnwrap, err := GetBaseFilesToUnwrap(filesMetaDto.Files, filesToUnwrap)
		if err != nil {
			return err
		}
		incrementFrom := NewBackup(folder.GetSubFolder(utility.BaseBackupPath), *sentinelDto.IncrementFrom)
		err = deltaFetchRecursionOld(incrementFrom, folder, dbDataDirectory, tablespaceSpec, baseFilesToUnwrap, extractProv)
		if err != nil {
			return err
		}
		tracelog.InfoLogger.Printf("%v fetched. Upgrading from LSN %s to LSN %s \n",
			*(sentinelDto.IncrementFrom),
			*(sentinelDto.IncrementFromLSN),
			*(sentinelDto.BackupStartLSN))
	}

	return backup.unwrapToEmptyDirectory(dbDataDirectory, filesToUnwrap, false, extractProv)
}

func GetPgFetcherOld(dbDataDirectory, fileMask, restoreSpecPath string,
	extractProv ExtractProvider,
) func(rootFolder storage.Folder, backup internal.Backup) {
	return func(rootFolder storage.Folder, backup internal.Backup) {
		pgBackup := ToPgBackup(backup)
		filesToUnwrap, err := pgBackup.GetFilesToUnwrap(fileMask)
		tracelog.ErrorLogger.FatalfOnError("Failed to fetch backup: %v\n", err)

		var spec *TablespaceSpec
		if restoreSpecPath != "" {
			spec = &TablespaceSpec{}
			err := readRestoreSpec(restoreSpecPath, spec)
			errMessege := fmt.Sprintf("Invalid restore specification path %s\n", restoreSpecPath)
			tracelog.ErrorLogger.FatalfOnError(errMessege, err)
		}

		err = deltaFetchRecursionOld(pgBackup, rootFolder, utility.ResolveSymlink(dbDataDirectory), spec, filesToUnwrap, extractProv)
		tracelog.ErrorLogger.FatalfOnError("Failed to fetch backup: %v\n", err)
	}
}

func GetBaseFilesToUnwrap(backupFileStates internal.BackupFileList, currentFilesToUnwrap map[string]bool) (map[string]bool, error) {
	baseFilesToUnwrap := make(map[string]bool)
	for file := range currentFilesToUnwrap {
		fileDescription, hasDescription := backupFileStates[file]
		if !hasDescription {
			if _, ok := UtilityFilePaths[file]; !ok {
				tracelog.ErrorLogger.Panicf("Wanted to fetch increment for file: '%s', but didn't find one in base", file)
			}
			continue
		}
		if fileDescription.IsSkipped || fileDescription.IsIncremented {
			baseFilesToUnwrap[file] = true
		}
	}
	return baseFilesToUnwrap, nil
}
