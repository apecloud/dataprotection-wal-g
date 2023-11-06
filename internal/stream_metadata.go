package internal

import (
	"errors"
	"io"

	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/spf13/viper"
	"github.com/wal-g/tracelog"
)

const (
	SplitMergeStreamBackup   = "SPLIT_MERGE_STREAM_BACKUP"
	SingleStreamStreamBackup = "STREAM_BACKUP"
)

type BackupStreamMetadata struct {
	Type        string `json:"type"`
	Partitions  uint   `json:"partitions,omitempty"`
	BlockSize   uint   `json:"block_size,omitempty"`
	Compression string `json:"compression,omitempty"`
}

func GetBackupStreamFetcher(backup Backup) (StreamFetcher, error) {
	var metadata BackupStreamMetadata
	err := FetchDto(backup.Folder, &metadata, StreamMetadataNameFromBackup(backup.Name))
	var test storage.ObjectNotFoundError
	if errors.As(err, &test) {
		return DownloadAndDecompressStream, nil
	}
	if err != nil {
		return nil, err
	}
	maxDownloadRetry := viper.GetInt(MysqlBackupDownloadMaxRetry)

	switch metadata.Type {
	case SplitMergeStreamBackup:
		var blockSize = metadata.BlockSize
		var compression = metadata.Compression
		return func(backup Backup, writer io.WriteCloser) error {
			return DownloadAndDecompressSplittedStream(backup, int(blockSize), compression, writer, maxDownloadRetry)
		}, nil
	case SingleStreamStreamBackup, "":
		return DownloadAndDecompressStream, nil
	}
	tracelog.ErrorLogger.Fatalf("Unknown backup type %s", metadata.Type)
	return nil, nil // unreachable
}

func UploadBackupStreamMetadata(uploader Uploader, metadata interface{}, backupName string) error {
	sentinelName := StreamMetadataNameFromBackup(backupName)
	return UploadDto(uploader.Folder(), metadata, sentinelName)
}
