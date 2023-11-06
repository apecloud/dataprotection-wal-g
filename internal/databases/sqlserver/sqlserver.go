package sqlserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/sqlserver/blob"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

const AllDatabases = "ALL"

const LogNamePrefix = "wal_"

const TimeSQLServerFormat = "Jan 02, 2006 03:04:05 PM"

const MaxTransferSize = 4 * 1024 * 1024

const MaxBlocksPerBlob = 25000 // 50000 actually, but we need some safety margin

const MaxBlobSize = MaxTransferSize * MaxBlocksPerBlob

const BlobNamePrefix = "blob_"

const ExternalBackupFilenameSeparator = ","

var SystemDbnames = []string{
	"master",
	"msdb",
	"model",
}

type SentinelDto struct {
	Server         string
	Databases      []string
	StartLocalTime time.Time `json:"StartLocalTime,omitempty"`
	StopLocalTime  time.Time `json:"StopLocalTime,omitempty"`
}

func (s *SentinelDto) String() string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

type DatabaseFile struct {
	LogicalName  string
	PhysicalName string
	Type         string
	FileID       int
}

func getSQLServerConnection() (*sql.DB, error) {
	connString, err := internal.GetRequiredSetting(internal.SQLServerConnectionString)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getDatabasesToBackup(db *sql.DB, dbnames []string) ([]string, error) {
	allDbnames, err := listDatabases(db)
	if err != nil {
		return nil, err
	}
	switch {
	case len(dbnames) == 1 && dbnames[0] == AllDatabases:
		return allDbnames, nil
	case len(dbnames) > 0:
		missing := exclude(dbnames, allDbnames)
		if len(missing) > 0 {
			return nil, fmt.Errorf("databases %v were not found in server", missing)
		}
		return dbnames, nil
	default:
		return exclude(allDbnames, SystemDbnames), nil
	}
}

func getDatabasesToRestore(sentinel *SentinelDto, dbnames []string, fromnames []string) ([]string, []string, error) {
	switch {
	case len(dbnames) == 0:
		if len(fromnames) != 0 {
			return nil, nil, fmt.Errorf("--from param should be omitted when --databases is empty")
		}
		dbnames = exclude(sentinel.Databases, SystemDbnames)
		fromnames = dbnames
	case len(dbnames) == 1 && dbnames[0] == AllDatabases:
		if len(fromnames) != 0 {
			return nil, nil, fmt.Errorf("--from param should be omitted when --databases %s", AllDatabases)
		}
		dbnames = sentinel.Databases
		fromnames = dbnames
	default:
		if len(fromnames) == 0 {
			fromnames = dbnames
		}
		if len(fromnames) != len(dbnames) {
			return nil, nil, fmt.Errorf("--from list length should match --databases list length")
		}
		missing := exclude(fromnames, sentinel.Databases)
		if len(missing) > 0 {
			return nil, nil, fmt.Errorf("databases %v were not found in backup", missing)
		}
	}
	return dbnames, fromnames, nil
}

func listDatabases(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sys.databases WHERE name != 'tempdb'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func estimateSize(db *sql.DB, query string, args ...interface{}) (int64, int, error) {
	var size int64
	row := db.QueryRow(query, args...)
	err := row.Scan(&size)
	if err != nil {
		return 0, 0, err
	}
	blobCount := int(math.Ceil(float64(size) / float64(MaxBlobSize)))
	return size, blobCount, nil
}

func estimateDBSize(db *sql.DB, dbname string) (int64, int, error) {
	query := fmt.Sprintf(`
		USE %s; 
		SELECT (SELECT SUM(used_log_space_in_bytes) FROM sys.dm_db_log_space_usage) 
			 + (SELECT SUM(allocated_extent_page_count)*8*1024 FROM sys.dm_db_file_space_usage)
		USE master;
	`, quoteName(dbname))
	return estimateSize(db, query)
}

func estimateLogSize(db *sql.DB, dbname string) (int64, int, error) {
	query := fmt.Sprintf(`
		USE %s; 
		SELECT SUM(log_space_in_bytes_since_last_backup) FROM sys.dm_db_log_space_usage; 
		USE master;
	`, quoteName(dbname))
	return estimateSize(db, query)
}

func listDatabaseFiles(db *sql.DB, urls string) ([]DatabaseFile, error) {
	var res []DatabaseFile
	query := fmt.Sprintf("RESTORE FILELISTONLY FROM %s", urls)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var dbf DatabaseFile
		err = utility.ScanToMap(rows, map[string]interface{}{
			"LogicalName":  &dbf.LogicalName,
			"PhysicalName": &dbf.PhysicalName,
			"Type":         &dbf.Type,
			"FileId":       &dbf.FileID,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, dbf)
	}
	return res, nil
}

func buildBackupUrlsList(baseURL string, blobCount int) []string {
	var res []string
	for i := 0; i < blobCount; i++ {
		res = append(res, fmt.Sprintf("%s/%s%03d", baseURL, BlobNamePrefix, i))
	}
	return res
}

func buildBackupUrls(baseURL string, blobCount int) string {
	res := ""
	for i, url := range buildBackupUrlsList(baseURL, blobCount) {
		if i > 0 {
			res += ", "
		}
		res += fmt.Sprintf("URL = '%s'", url)
	}
	return res
}

func buildRestoreUrlsList(baseURL string, blobNames []string) []string {
	if len(blobNames) == 0 {
		// old-style single blob backup
		return []string{baseURL}
	}
	var res []string
	for _, blobName := range blobNames {
		res = append(res, fmt.Sprintf("%s/%s", baseURL, blobName))
	}
	return res
}

func buildRestoreUrls(baseURL string, blobNames []string) string {
	res := ""
	for i, url := range buildRestoreUrlsList(baseURL, blobNames) {
		if i > 0 {
			res += ", "
		}
		res += fmt.Sprintf("URL = '%s'", url)
	}
	return res
}

func buildPhysicalFileMove(files []DatabaseFile, dbname string, datadir string, logdir string) (string, error) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].FileID < files[j].FileID
	})
	res := ""
	dataFileCnt := 0
	logFileCnt := 0
	for _, file := range files {
		var newName string
		var newPhysicalName string
		switch file.Type {
		case "D":
			suffix := ""
			if dataFileCnt > 0 {
				suffix = fmt.Sprintf("_%d", dataFileCnt)
			}
			dataFileCnt++
			ext := ".mdf"
			if file.FileID > 1 {
				ext = ".ndf"
			}
			newName = dbname + suffix + ext
			newPhysicalName = filepath.Join(datadir, newName)
		case "L":
			suffix := "_log"
			if logFileCnt > 0 {
				suffix = fmt.Sprintf("_log_%d", logFileCnt)
			}
			logFileCnt++
			newName = dbname + suffix + ".ldf"
			newPhysicalName = filepath.Join(logdir, newName)
		default:
			return "", fmt.Errorf("unexpected backup file type: '%s'", file.Type)
		}
		if res != "" {
			res += ", "
		}
		res += fmt.Sprintf("MOVE %s TO %s", quoteValue(file.LogicalName), quoteValue(newPhysicalName))
	}
	return res, nil
}

func quoteName(name string) string {
	return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
}

func quoteValue(val string) string {
	return "'" + strings.ReplaceAll(val, "'", "''") + "'"
}

func generateDatabaseBackupName() string {
	return utility.BackupNamePrefix + utility.TimeNowCrossPlatformUTC().Format(utility.BackupTimeFormat)
}

func getDatabaseBackupPath(backupName, dbname string) string {
	return path.Join(utility.BaseBackupPath, backupName, dbname)
}

func getDatabaseBackupURL(backupName, dbname string) string {
	hostname, err := internal.GetRequiredSetting(internal.SQLServerBlobHostname)
	if err != nil {
		tracelog.ErrorLogger.FatalOnError(err)
	}
	backupName = url.QueryEscape(backupName)
	dbname = url.QueryEscape(dbname)
	return fmt.Sprintf("https://%s/%s", hostname, getDatabaseBackupPath(backupName, dbname))
}

func generateLogBackupName() string {
	return LogNamePrefix + utility.TimeNowCrossPlatformUTC().Format(utility.BackupTimeFormat)
}

func getLogBackupPath(logBackupName, dbname string) string {
	return path.Join(utility.WalPath, logBackupName, dbname)
}

func getLogBackupURL(logBackupName, dbname string) string {
	hostname, err := internal.GetRequiredSetting(internal.SQLServerBlobHostname)
	if err != nil {
		tracelog.ErrorLogger.FatalOnError(err)
	}
	logBackupName = url.QueryEscape(logBackupName)
	dbname = url.QueryEscape(dbname)
	return fmt.Sprintf("https://%s/%s", hostname, getLogBackupPath(logBackupName, dbname))
}

func doesLogBackupContainDB(folder storage.Folder, logBakupName string, dbname string) (bool, error) {
	f := folder.GetSubFolder(utility.WalPath).GetSubFolder(logBakupName)
	_, dbDirs, err := f.ListFolder()
	if err != nil {
		return false, err
	}
	for _, dbDir := range dbDirs {
		if dbname == path.Base(dbDir.GetPath()) {
			return true, nil
		}
	}
	return false, nil
}

func listBackupBlobs(folder storage.Folder) ([]string, error) {
	ok, err := folder.Exists(blob.IndexFileName)
	if err != nil {
		return nil, err
	}
	if ok {
		// old-style single blob backup
		return nil, nil
	}
	_, blobDirs, err := folder.ListFolder()
	if err != nil {
		return nil, err
	}
	var blobs []string
	for _, blobDir := range blobDirs {
		name := path.Base(blobDir.GetPath())
		if strings.HasPrefix(name, BlobNamePrefix) {
			blobs = append(blobs, name)
		}
	}
	sort.Strings(blobs)
	return blobs, nil
}

func getLogsSinceBackup(folder storage.Folder, backupName string, stopAt time.Time) ([]string, error) {
	if !strings.HasPrefix(backupName, utility.BackupNamePrefix) {
		return nil, fmt.Errorf("unexpected backup name: %s", backupName)
	}
	startTS := backupName[len(utility.BackupNamePrefix):]
	endTS := stopAt.Format(utility.BackupTimeFormat)
	_, logBackups, err := folder.GetSubFolder(utility.WalPath).ListFolder()
	if err != nil {
		return nil, err
	}
	var allLogNames []string
	for _, logBackup := range logBackups {
		allLogNames = append(allLogNames, path.Base(logBackup.GetPath()))
	}
	sort.Strings(allLogNames)

	var logNames []string
	for _, name := range allLogNames {
		logTS := name[len(LogNamePrefix):]
		if logTS < startTS {
			continue
		}
		logNames = append(logNames, name)
		if logTS > endTS {
			break
		}
	}

	return logNames, nil
}

func runParallel(f func(int) error, cnt int, concurrency int) error {
	if concurrency <= 0 {
		concurrency = cnt
	}
	sem := make(chan struct{}, concurrency)
	errs := make(chan error, cnt)
	for i := 0; i < cnt; i++ {
		go func(i int) {
			sem <- struct{}{}
			defer func() { <-sem }()
			errs <- f(i)
		}(i)
	}
	var errStr string
	for i := 0; i < cnt; i++ {
		err := <-errs
		if err != nil {
			errStr += err.Error() + "\n"
		}
	}
	if errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

func getDBConcurrency() int {
	concurrency, err := internal.GetMaxConcurrency(internal.SQLServerDBConcurrency)
	if err != nil {
		tracelog.WarningLogger.Printf("config error: %v", err)
		tracelog.WarningLogger.Printf("using default db concurrency: %d", blob.DefaultConcurrency)
		return blob.DefaultConcurrency
	}
	return concurrency
}

func exclude(src, excl []string) []string {
	var res []string
SRC:
	for _, r := range src {
		for _, r2 := range excl {
			if r2 == r {
				continue SRC
			}
		}
		res = append(res, r)
	}
	return res
}

func uniq(src []string) []string {
	res := make([]string, 0, len(src))
	done := make(map[string]struct{}, len(src))
	for _, s := range src {
		if _, ok := done[s]; !ok {
			res = append(res, s)
			done[s] = struct{}{}
		}
	}
	return res
}

type BackupProperties struct {
	BackupType        int
	DatabaseName      string
	FirstLSN          string
	LastLSN           string
	CheckpointLSN     string
	DatabaseBackupLSN string
	BackupStartDate   time.Time
	BackupFinishDate  time.Time
	HasBulkLoggedData bool
	IsSnapshot        bool
	IsReadOnly        bool
	IsSingleUser      bool
	BackupURL         string
	BackupFile        string
}

func GetBackupProperties(db *sql.DB,
	folder storage.Folder,
	logBackup bool,
	backupName string,
	databaseName string,
) ([]*BackupProperties, error) {
	var res []*BackupProperties
	var baseURL string
	var basePath string
	if logBackup {
		baseURL = getLogBackupURL(backupName, databaseName)
		basePath = getLogBackupPath(backupName, databaseName)
	} else {
		baseURL = getDatabaseBackupURL(backupName, databaseName)
		basePath = getDatabaseBackupPath(backupName, databaseName)
	}
	blobs, err := listBackupBlobs(folder.GetSubFolder(basePath))
	if err != nil {
		return res, err
	}
	urls := buildRestoreUrls(baseURL, blobs)
	query := fmt.Sprintf("RESTORE HEADERONLY FROM %s", urls)
	rows, err := db.Query(query)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var dbf BackupProperties
		err = utility.ScanToMap(rows, map[string]interface{}{
			"BackupType":        &dbf.BackupType,
			"DatabaseName":      &dbf.DatabaseName,
			"FirstLSN":          &dbf.FirstLSN,
			"LastLSN":           &dbf.LastLSN,
			"CheckpointLSN":     &dbf.CheckpointLSN,
			"DatabaseBackupLSN": &dbf.DatabaseBackupLSN,
			"BackupStartDate":   &dbf.BackupStartDate,
			"BackupFinishDate":  &dbf.BackupFinishDate,
			"HasBulkLoggedData": &dbf.HasBulkLoggedData,
			"IsSnapshot":        &dbf.IsSnapshot,
			"IsReadOnly":        &dbf.IsReadOnly,
			"IsSingleUser":      &dbf.IsSingleUser,
		})
		if err != nil {
			return nil, err
		}
		dbf.BackupURL = urls
		dbf.BackupFile = backupName
		res = append(res, &dbf)
	}
	return res, nil
}

type LockWrapper struct {
	c io.Closer
}

func (lw LockWrapper) Close() {
	if lw.c != nil {
		tracelog.ErrorLogger.PrintOnError(lw.c.Close())
	}
}

func RunOrReuseProxy(ctx context.Context, cancel context.CancelFunc, folder storage.Folder) (*LockWrapper, error) {
	bs, err := blob.NewServer(folder)
	if err != nil {
		return nil, xerrors.Errorf("proxy create error: %v", err)
	}
	reuse, _, err := internal.GetBoolSetting(internal.SQLServerReuseProxy)
	if err != nil {
		return nil, err
	}
	if reuse {
		return &LockWrapper{nil}, bs.WaitReady(ctx, blob.ProxyStartTimeout)
	}
	lock, err := bs.AcquireLock()
	if err != nil {
		return nil, err
	}

	err = bs.RunBackground(ctx, cancel)
	if err != nil {
		tracelog.ErrorLogger.PrintOnError(lock.Close())
		return nil, xerrors.Errorf("proxy run error: %v", err)
	}
	return &LockWrapper{lock}, nil
}

func GetDBRestoreLSN(db *sql.DB, databaseName string) (string, error) {
	query := `SELECT MAX(redo_start_lsn) 
        FROM sys.master_files
        WHERE database_id=DB_ID(@dbname) `
	var res string
	if err := db.QueryRow(query, sql.Named("dbname", databaseName)).Scan(&res); err != nil {
		return "0", err
	}
	return res, nil
}

func IsLogAlreadyApplied(db *sql.DB, databaseName string, logBackupFileProperties *BackupProperties) (bool, error) {
	dbRestoreLSN, err := GetDBRestoreLSN(db, databaseName)
	if err != nil {
		return false, err
	}
	dbRestoreLSNInt := new(big.Int)
	dbRestoreLSNInt, ok := dbRestoreLSNInt.SetString(dbRestoreLSN, 10)
	if !ok {
		return false, xerrors.Errorf("dbRestoreLSN not recognized")
	}
	lastLSNInt := new(big.Int)
	lastLSNInt, ok = lastLSNInt.SetString(logBackupFileProperties.LastLSN, 10)
	if !ok {
		return false, xerrors.Errorf("lastLSN not recognized")
	}
	if dbRestoreLSNInt.Cmp(lastLSNInt) == -1 {
		return false, nil
	}
	return true, nil
}

func GetDefaultDataLogDirs(db *sql.DB) (string, string, error) {
	var datadir, logdir string
	query := `SELECT serverproperty('InstanceDefaultDataPath'), serverproperty('InstanceDefaultLogPath')`
	err := db.QueryRow(query).Scan(&datadir, &logdir)
	return strings.TrimSpace(datadir), strings.TrimSpace(logdir), err
}
