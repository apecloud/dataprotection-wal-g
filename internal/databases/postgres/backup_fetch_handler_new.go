package postgres

import (
	"fmt"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

func GetPgFetcherNew(dbDataDirectory, fileMask, restoreSpecPath string, skipRedundantTars bool,
	extractProv ExtractProvider,
) func(folder storage.Folder, backup internal.Backup) {
	return func(folder storage.Folder, backup internal.Backup) {
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

		// directory must be empty before starting a deltaFetch
		isEmpty, err := utility.IsDirectoryEmpty(dbDataDirectory)
		tracelog.ErrorLogger.FatalfOnError("Failed to fetch backup: %v\n", err)

		if !isEmpty {
			tracelog.ErrorLogger.FatalfOnError("Failed to fetch backup: %v\n",
				NewNonEmptyDBDataDirectoryError(dbDataDirectory))
		}
		config := NewFetchConfig(pgBackup.Name,
			utility.ResolveSymlink(dbDataDirectory), folder, spec, filesToUnwrap, skipRedundantTars, extractProv)
		err = deltaFetchRecursionNew(config)
		tracelog.ErrorLogger.FatalfOnError("Failed to fetch backup: %v\n", err)
	}
}

// TODO : unit tests
// deltaFetchRecursion function composes Backup object and recursively searches for necessary base backup
func deltaFetchRecursionNew(cfg *FetchConfig) error {
	backup := NewBackup(cfg.folder.GetSubFolder(utility.BaseBackupPath), cfg.backupName)
	sentinelDto, filesMetaDto, err := backup.GetSentinelAndFilesMetadata()
	if err != nil {
		return err
	}
	cfg.tablespaceSpec = chooseTablespaceSpecification(sentinelDto.TablespaceSpec, cfg.tablespaceSpec)
	sentinelDto.TablespaceSpec = cfg.tablespaceSpec

	if sentinelDto.IsIncremental() {
		tracelog.InfoLogger.Printf("Delta %v at LSN %s \n",
			cfg.backupName,
			*(sentinelDto.BackupStartLSN))
		baseFilesToUnwrap, err := GetBaseFilesToUnwrap(filesMetaDto.Files, cfg.filesToUnwrap)
		if err != nil {
			return err
		}
		unwrapResult, err := backup.unwrapNew(cfg.dbDataDirectory, cfg.filesToUnwrap,
			false, cfg.skipRedundantTars, cfg.extractProv)
		if err != nil {
			return err
		}
		cfg.filesToUnwrap = baseFilesToUnwrap
		cfg.backupName = *sentinelDto.IncrementFrom
		if cfg.skipRedundantTars {
			// if we skip redundant tars we should exclude files that
			// no longer need any additional information (completed ones)
			cfg.SkipRedundantFiles(unwrapResult)
		}
		tracelog.InfoLogger.Printf("%v fetched. Downgrading from LSN %s to LSN %s \n",
			cfg.backupName,
			*(sentinelDto.BackupStartLSN),
			*(sentinelDto.IncrementFromLSN))
		err = deltaFetchRecursionNew(cfg)
		if err != nil {
			return err
		}

		return nil
	}

	tracelog.InfoLogger.Printf("%s reached. Applying base backup... \n",
		*(sentinelDto.BackupStartLSN))
	_, err = backup.unwrapNew(cfg.dbDataDirectory, cfg.filesToUnwrap,
		false, cfg.skipRedundantTars, cfg.extractProv)
	return err
}
