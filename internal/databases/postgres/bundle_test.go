package postgres_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/compression"
	"github.com/apecloud/dataprotection-wal-g/internal/compression/lz4"
	"github.com/apecloud/dataprotection-wal-g/internal/databases/postgres"
	"github.com/apecloud/dataprotection-wal-g/internal/walparser"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/memory"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/testtools"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var BundleTestLocations = []walparser.BlockLocation{
	*walparser.NewBlockLocation(1, 2, 3, 4),
	*walparser.NewBlockLocation(5, 6, 7, 8),
	*walparser.NewBlockLocation(1, 2, 3, 9),
}

func TestEmptyBundleQueue(t *testing.T) {
	internal.ConfigureSettings(internal.PG)
	internal.InitConfig()
	internal.Configure()

	bundle := &postgres.Bundle{
		Bundle: internal.Bundle{
			Directory:        "",
			TarSizeThreshold: 100,
		},
	}

	uploader := testtools.NewMockUploader(false, false)
	tarBallMaker := internal.NewStorageTarBallMaker("mockBackup", uploader)

	err := bundle.StartQueue(tarBallMaker)
	assert.NoError(t, err)

	err = bundle.FinishQueue()
	assert.NoError(t, err)
}

func TestBundleQueue(t *testing.T) {
	queueTest(t)
}

func TestBundleQueueHighConcurrency(t *testing.T) {
	viper.Set(internal.UploadConcurrencySetting, "100")
	queueTest(t)
}

func TestBundleQueueLowConcurrency(t *testing.T) {
	viper.Set(internal.UploadConcurrencySetting, "1")
	queueTest(t)
}

func queueTest(t *testing.T) {
	bundle := &postgres.Bundle{
		Bundle: internal.Bundle{
			Directory:        "",
			TarSizeThreshold: 100,
		},
	}
	uploader := testtools.NewMockUploader(false, false)
	tarBallMaker := internal.NewStorageTarBallMaker("mockBackup", uploader)

	// For tests there must be at least 3 workers

	bundle.StartQueue(tarBallMaker)

	a := bundle.TarBallQueue.Deque()
	go func() {
		time.Sleep(10 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		bundle.TarBallQueue.EnqueueBack(a)
	}()

	c := bundle.TarBallQueue.Deque()
	go func() {
		time.Sleep(10 * time.Millisecond)
		bundle.TarBallQueue.CheckSizeAndEnqueueBack(c)
	}()

	b := bundle.TarBallQueue.Deque()
	go func() {
		time.Sleep(10 * time.Millisecond)
		bundle.TarBallQueue.EnqueueBack(b)
	}()

	err := bundle.FinishQueue()
	if err != nil {
		t.Log(err)
	}
}

func makeDeltaFile(locations []walparser.BlockLocation) ([]byte, error) {
	locations = append(locations, walparser.TerminalLocation)
	var data bytes.Buffer
	compressor := compression.Compressors[lz4.AlgorithmName]
	compressingWriter := compressor.NewWriter(&data)
	err := walparser.WriteLocationsTo(compressingWriter, locations)
	if err != nil {
		return nil, err
	}
	_, err = compressingWriter.Write([]byte{0, 0, 0, 0})
	if err != nil {
		return nil, err
	}
	err = compressingWriter.Close()
	if err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

func putDeltaIntoStorage(storage *memory.Storage, locations []walparser.BlockLocation, deltaFilename string) error {
	deltaData, err := makeDeltaFile(locations)
	if err != nil {
		return err
	}
	storage.Store("in_memory/wal_005/"+deltaFilename+".lz4", *bytes.NewBuffer(deltaData))
	return nil
}

func putWalIntoStorage(storage *memory.Storage, data []byte, walFilename string) error {
	compressor := compression.Compressors[lz4.AlgorithmName]
	var compressedData bytes.Buffer
	compressingWriter := compressor.NewWriter(&compressedData)
	_, err := utility.FastCopy(compressingWriter, bytes.NewReader(data))
	if err != nil {
		return err
	}
	err = compressingWriter.Close()
	if err != nil {
		return err
	}
	storage.Store("in_memory/wal_005/"+walFilename+".lz4", compressedData)
	return nil
}

func fillStorageWithMockDeltas(storage *memory.Storage) error {
	err := putDeltaIntoStorage(
		storage,
		[]walparser.BlockLocation{
			BundleTestLocations[0],
			BundleTestLocations[1],
		},
		"000000010000000000000070_delta",
	)
	if err != nil {
		return err
	}
	err = putDeltaIntoStorage(
		storage,
		[]walparser.BlockLocation{
			BundleTestLocations[0],
			BundleTestLocations[2],
		},
		"000000010000000000000080_delta",
	)
	if err != nil {
		return err
	}
	err = putDeltaIntoStorage(
		storage,
		[]walparser.BlockLocation{
			BundleTestLocations[2],
		},
		"0000000100000000000000A0_delta",
	)
	if err != nil {
		return err
	}
	err = putWalIntoStorage(storage, testtools.CreateWalPageWithContinuation(), "000000010000000000000090")
	return err
}

func setupFolderAndBundle() (folder storage.Folder, bundle *postgres.Bundle, err error) {
	storage := memory.NewStorage()
	err = fillStorageWithMockDeltas(storage)
	if err != nil {
		return nil, nil, err
	}
	folder = memory.NewFolder("in_memory/", storage).GetSubFolder(utility.WalPath)
	currentBackupFirstWalFilename := "000000010000000000000073"
	timeLine, logSegNo, err := postgres.ParseWALFilename(currentBackupFirstWalFilename)
	if err != nil {
		return nil, nil, err
	}
	incrementFromLsn := postgres.LSN(logSegNo * postgres.WalSegmentSize)
	bundle = &postgres.Bundle{
		Timeline:         timeLine,
		IncrementFromLsn: &incrementFromLsn,
	}
	return
}

func TestLoadDeltaMap_AllDeltas(t *testing.T) {
	folder, bundle, err := setupFolderAndBundle()
	assert.NoError(t, err)

	backupNextWalFilename := "000000010000000000000090"
	_, curLogSegNo, _ := postgres.ParseWALFilename(backupNextWalFilename)

	backupStartLsn := postgres.LSN(curLogSegNo*postgres.WalSegmentSize + 1)
	err = bundle.DownloadDeltaMap(internal.NewFolderReader(folder), backupStartLsn)
	deltaMap := bundle.DeltaMap
	assert.NoError(t, err)
	assert.NotNil(t, deltaMap)
	assert.Contains(t, deltaMap, BundleTestLocations[0].RelationFileNode)
	assert.Contains(t, deltaMap, BundleTestLocations[1].RelationFileNode)
	assert.Equal(t, []uint32{4, 9}, deltaMap[BundleTestLocations[0].RelationFileNode].ToArray())
	assert.Equal(t, []uint32{8}, deltaMap[BundleTestLocations[1].RelationFileNode].ToArray())
}

func TestLoadDeltaMap_MissingDelta(t *testing.T) {
	folder, bundle, err := setupFolderAndBundle()
	assert.NoError(t, err)

	backupNextWalFilename := "0000000100000000000000B0"
	_, curLogSegNo, _ := postgres.ParseWALFilename(backupNextWalFilename)

	err = bundle.DownloadDeltaMap(internal.NewFolderReader(folder), postgres.LSN(curLogSegNo*postgres.WalSegmentSize))
	assert.Error(t, err)
	assert.Nil(t, bundle.DeltaMap)
}

func TestLoadDeltaMap_WalTail(t *testing.T) {
	folder, bundle, err := setupFolderAndBundle()
	assert.NoError(t, err)

	backupNextWalFilename := "000000010000000000000091"
	_, curLogSegNo, _ := postgres.ParseWALFilename(backupNextWalFilename)

	err = bundle.DownloadDeltaMap(internal.NewFolderReader(folder), postgres.LSN(curLogSegNo*postgres.WalSegmentSize))
	assert.NoError(t, err)
	assert.NotNil(t, bundle.DeltaMap)
	assert.Equal(t, []uint32{4, 9}, bundle.DeltaMap[BundleTestLocations[0].RelationFileNode].ToArray())
	assert.Equal(t, []uint32{8}, bundle.DeltaMap[BundleTestLocations[1].RelationFileNode].ToArray())
}
