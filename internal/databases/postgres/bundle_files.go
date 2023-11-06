package postgres

import (
	"archive/tar"
	"os"
	"sync"

	"github.com/apecloud/dataprotection-wal-g/internal"

	"github.com/apecloud/dataprotection-wal-g/internal/walparser"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

func newStatBundleFiles(fileStat RelFileStatistics) *StatBundleFiles {
	return &StatBundleFiles{fileStats: fileStat}
}

// StatBundleFiles contains the bundle files.
// Additionally, it calculates and stores the updates count for each added file
type StatBundleFiles struct {
	sync.Map
	fileStats RelFileStatistics
}

func (files *StatBundleFiles) AddFileWithCorruptBlocks(tarHeader *tar.Header,
	fileInfo os.FileInfo,
	isIncremented bool,
	corruptedBlocks []uint32,
	storeAllBlocks bool) {
	updatesCount := files.fileStats.getFileUpdateCount(tarHeader.Name)
	fileDescription := internal.BackupFileDescription{IsSkipped: false, IsIncremented: isIncremented, MTime: fileInfo.ModTime(),
		UpdatesCount: updatesCount}
	fileDescription.SetCorruptBlocks(corruptedBlocks, storeAllBlocks)
	files.AddFileDescription(tarHeader.Name, fileDescription)
}

func (files *StatBundleFiles) AddSkippedFile(tarHeader *tar.Header, fileInfo os.FileInfo) {
	updatesCount := files.fileStats.getFileUpdateCount(tarHeader.Name)
	files.AddFileDescription(tarHeader.Name,
		internal.BackupFileDescription{IsSkipped: true, IsIncremented: false,
			MTime: fileInfo.ModTime(), UpdatesCount: updatesCount})
}

func (files *StatBundleFiles) AddFile(tarHeader *tar.Header, fileInfo os.FileInfo, isIncremented bool) {
	updatesCount := files.fileStats.getFileUpdateCount(tarHeader.Name)
	files.AddFileDescription(tarHeader.Name,
		internal.BackupFileDescription{IsSkipped: false, IsIncremented: isIncremented,
			MTime: fileInfo.ModTime(), UpdatesCount: updatesCount})
}

func (files *StatBundleFiles) AddFileDescription(name string, backupFileDescription internal.BackupFileDescription) {
	files.Store(name, backupFileDescription)
}

func (files *StatBundleFiles) GetUnderlyingMap() *sync.Map {
	return &files.Map
}

type RelFileStatistics map[walparser.RelFileNode]PgRelationStat

func (relStat *RelFileStatistics) getFileUpdateCount(filePath string) uint64 {
	if relStat == nil {
		return 0
	}
	relFileNode, err := GetRelFileNodeFrom(filePath)
	if err != nil {
		return 0
	}
	fileStat, ok := (*relStat)[*relFileNode]
	if !ok {
		return 0
	}
	return fileStat.deletedTuplesCount + fileStat.updatedTuplesCount + fileStat.insertedTuplesCount
}

func newRelFileStatistics(queryRunner *PgQueryRunner) (RelFileStatistics, error) {
	databases, err := queryRunner.GetDatabaseInfos()
	if err != nil {
		return nil, errors.Wrap(err, "CollectStatistics: Failed to get db names.")
	}

	result := make(map[walparser.RelFileNode]PgRelationStat)
	// CollectStatistics collects statistics for each relFileNode
	for _, db := range databases {
		dbName := db.Name
		databaseOption := func(c *pgx.ConnConfig) error {
			c.Database = dbName
			return nil
		}
		dbConn, err := Connect(databaseOption)
		if err != nil {
			tracelog.WarningLogger.Printf("Failed to collect statistics for database: %s\n'%v'\n", db.Name, err)
			continue
		}

		queryRunner, err := NewPgQueryRunner(dbConn)
		if err != nil {
			return nil, errors.Wrap(err, "CollectStatistics: Failed to build query runner.")
		}
		pgStatRows, err := queryRunner.getStatistics(db)
		if err != nil {
			return nil, errors.Wrap(err, "CollectStatistics: Failed to collect statistics.")
		}
		for relFileNode, statRow := range pgStatRows {
			result[relFileNode] = statRow
		}
		err = dbConn.Close()
		tracelog.WarningLogger.PrintOnError(err)
	}
	return result, nil
}
