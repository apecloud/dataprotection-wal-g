package postgres

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/apecloud/dataprotection-wal-g/internal"

	"github.com/apecloud/dataprotection-wal-g/internal/fsutil"
	"github.com/apecloud/dataprotection-wal-g/internal/ioextensions"
	"github.com/apecloud/dataprotection-wal-g/internal/walparser"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

type DeltaFileWriterNotFoundError struct {
	error
}

func newDeltaFileWriterNotFoundError(filename string) DeltaFileWriterNotFoundError {
	return DeltaFileWriterNotFoundError{errors.Errorf("can't file delta file writer for file: '%s'", filename)}
}

func (err DeltaFileWriterNotFoundError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

type DeltaFileManager struct {
	dataFolder            fsutil.DataFolder
	PartFiles             *internal.LazyCache[string, *WalPartFile]
	DeltaFileWriters      *internal.LazyCache[string, *DeltaFileChanWriter]
	deltaFileWriterWaiter sync.WaitGroup
	canceledWalRecordings chan string
	CanceledDeltaFiles    map[string]bool
	canceledWaiter        sync.WaitGroup
}

func NewDeltaFileManager(dataFolder fsutil.DataFolder) *DeltaFileManager {
	manager := &DeltaFileManager{
		dataFolder:            dataFolder,
		canceledWalRecordings: make(chan string),
		CanceledDeltaFiles:    make(map[string]bool),
	}
	manager.PartFiles = internal.NewLazyCache(
		func(partFilename string) (partFile *WalPartFile, err error) {
			return manager.loadPartFile(partFilename)
		})
	manager.DeltaFileWriters = internal.NewLazyCache(
		func(deltaFilename string) (deltaFileWriter *DeltaFileChanWriter, err error) {
			return manager.loadDeltaFileWriter(deltaFilename)
		})
	manager.canceledWaiter.Add(1)
	go manager.collectCanceledDeltaFiles()
	return manager
}

func (manager *DeltaFileManager) GetBlockLocationConsumer(deltaFilename string) (chan walparser.BlockLocation, error) {
	deltaFileWriter, _, err := manager.DeltaFileWriters.Load(deltaFilename)
	if err != nil {
		return nil, err
	}
	return deltaFileWriter.BlockLocationConsumer, nil
}

// TODO : unit tests
func (manager *DeltaFileManager) loadDeltaFileWriter(
	deltaFilename string) (deltaFileWriter *DeltaFileChanWriter, err error) {
	physicalDeltaFile, err := manager.dataFolder.OpenReadonlyFile(deltaFilename)
	var deltaFile *DeltaFile
	if err != nil {
		if _, ok := err.(fsutil.NoSuchFileError); !ok {
			return nil, err
		}
		deltaFile, err = NewDeltaFile(walparser.NewWalParser())
		if err != nil {
			return nil, err
		}
	} else {
		defer utility.LoggedClose(physicalDeltaFile, "")
		deltaFile, err = LoadDeltaFile(physicalDeltaFile)
		if err != nil {
			return nil, err
		}
	}
	deltaFileWriter = NewDeltaFileChanWriter(deltaFile)
	manager.deltaFileWriterWaiter.Add(1)
	go deltaFileWriter.Consume(&manager.deltaFileWriterWaiter)
	return deltaFileWriter, nil
}

func (manager *DeltaFileManager) GetPartFile(deltaFilename string) (*WalPartFile, error) {
	partFilename := ToPartFilename(deltaFilename)
	partFile, _, err := manager.PartFiles.Load(partFilename)
	if err != nil {
		return nil, err
	}
	return partFile, nil
}

// TODO : unit tests
func (manager *DeltaFileManager) loadPartFile(partFilename string) (*WalPartFile, error) {
	physicalPartFile, err := manager.dataFolder.OpenReadonlyFile(partFilename)
	var partFile *WalPartFile
	if err != nil {
		if _, ok := err.(fsutil.NoSuchFileError); !ok {
			return nil, err
		}
		partFile = NewWalPartFile()
	} else {
		defer utility.LoggedClose(physicalPartFile, "")
		partFile, err = LoadPartFile(physicalPartFile)
		if err != nil {
			return nil, err
		}
	}
	return partFile, nil
}

func (manager *DeltaFileManager) FlushPartFiles() (completedPartFiles map[string]bool) {
	close(manager.canceledWalRecordings)
	manager.canceledWaiter.Wait()
	completedPartFiles = make(map[string]bool)
	manager.PartFiles.Range(func(partFilename string, partFile *WalPartFile) bool {
		deltaFilename := partFilenameToDelta(partFilename)
		if _, ok := manager.CanceledDeltaFiles[deltaFilename]; ok {
			return true
		}
		if partFile.IsComplete() {
			completedPartFiles[partFilename] = true
			err := manager.CombinePartFile(deltaFilename, partFile)
			if err != nil {
				manager.CanceledDeltaFiles[deltaFilename] = true
				tracelog.WarningLogger.Printf(
					"Canceled delta file writing because of error: "+tracelog.GetErrorFormatter()+"\n",
					err)
			}
		} else {
			err := fsutil.SaveToDataFolder(partFile, partFilename, manager.dataFolder)
			if err != nil {
				manager.CanceledDeltaFiles[deltaFilename] = true
				tracelog.WarningLogger.Printf("Failed to save part file: '%s' because of error: '%v'\n", partFilename, err)
			}
		}
		return true
	})
	return
}

func (manager *DeltaFileManager) FlushDeltaFiles(uploader internal.Uploader, completedPartFiles map[string]bool) {
	manager.DeltaFileWriters.Range(func(key string, deltaFileWriter *DeltaFileChanWriter) bool {
		deltaFileWriter.close()
		return true
	})
	manager.deltaFileWriterWaiter.Wait()
	manager.DeltaFileWriters.Range(func(deltaFilename string, deltaFileWriter *DeltaFileChanWriter) bool {
		if _, ok := manager.CanceledDeltaFiles[deltaFilename]; ok {
			return true
		}
		partFilename := ToPartFilename(deltaFilename)
		if _, ok := completedPartFiles[partFilename]; ok {
			var deltaFileData bytes.Buffer
			err := deltaFileWriter.DeltaFile.Save(&deltaFileData)
			if err != nil {
				tracelog.WarningLogger.Printf(
					"Failed to upload delta file: '%s' because of saving error: '%v'\n",
					deltaFilename, err)
			} else {
				err = uploader.UploadFile(ioextensions.NewNamedReaderImpl(&deltaFileData, deltaFilename))
				if err != nil {
					tracelog.WarningLogger.Printf(
						"Failed to upload delta file: '%s' because of uploading error: '%v'\n",
						deltaFilename, err)
				}
			}
		} else {
			err := fsutil.SaveToDataFolder(deltaFileWriter.DeltaFile, deltaFilename, manager.dataFolder)
			if err != nil {
				tracelog.WarningLogger.Printf("Failed to save delta file: '%s' because of error: '%v'\n", deltaFilename, err)
			}
		}
		return true
	})
}

func (manager *DeltaFileManager) FlushFiles(uploader internal.Uploader) {
	err := manager.dataFolder.CleanFolder()
	if err != nil {
		tracelog.WarningLogger.Printf("Failed to clean delta folder because of error: '%v'\n", err)
	}
	completedPartFiles := manager.FlushPartFiles()
	manager.FlushDeltaFiles(uploader, completedPartFiles)
}

func (manager *DeltaFileManager) CancelRecording(walFilename string) {
	manager.canceledWalRecordings <- walFilename
}

// TODO : unit tests
func (manager *DeltaFileManager) collectCanceledDeltaFiles() {
	for walFilename := range manager.canceledWalRecordings {
		deltaFilename, err := GetDeltaFilenameFor(walFilename)
		if err != nil {
			continue
		}
		manager.CanceledDeltaFiles[deltaFilename] = true
		nextWalFilename, _ := GetNextWalFilename(walFilename)
		deltaFilename, _ = GetDeltaFilenameFor(nextWalFilename)
		manager.CanceledDeltaFiles[deltaFilename] = true
	}
	manager.canceledWaiter.Done()
}

func (manager *DeltaFileManager) CombinePartFile(deltaFilename string, partFile *WalPartFile) error {
	deltaFileWriter, exists := manager.DeltaFileWriters.LoadExisting(deltaFilename)
	if !exists {
		return newDeltaFileWriterNotFoundError(deltaFilename)
	}
	deltaFileWriter.DeltaFile.WalParser = walparser.LoadWalParserFromCurrentRecordHead(partFile.WalHeads[WalFileInDelta-1])
	records, err := partFile.CombineRecords()
	if err != nil {
		return err
	}
	locations := walparser.ExtractBlockLocations(records)
	for _, location := range locations {
		deltaFileWriter.BlockLocationConsumer <- location
	}
	return nil
}
