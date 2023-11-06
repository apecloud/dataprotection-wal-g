package postgres

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/apecloud/dataprotection-wal-g/internal"

	"github.com/RoaringBitmap/roaring"
	"github.com/apecloud/dataprotection-wal-g/internal/ioextensions"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
	"golang.org/x/sync/errgroup"
)

type SkippedFileError struct {
	error
}

func newSkippedFileError(path string) SkippedFileError {
	return SkippedFileError{errors.Errorf("File is skipped from the current backup: %s\n", path)}
}

func (err SkippedFileError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

type TarBallFilePackerOptions struct {
	verifyPageChecksums   bool
	storeAllCorruptBlocks bool
}

func NewTarBallFilePackerOptions(verifyPageChecksums, storeAllCorruptBlocks bool) TarBallFilePackerOptions {
	return TarBallFilePackerOptions{
		verifyPageChecksums:   verifyPageChecksums,
		storeAllCorruptBlocks: storeAllCorruptBlocks,
	}
}

// TarBallFilePackerImpl is used to pack bundle file into tarball.
type TarBallFilePackerImpl struct {
	deltaMap         PagedFileDeltaMap
	incrementFromLsn *LSN
	files            internal.BundleFiles
	options          TarBallFilePackerOptions
}

func NewTarBallFilePacker(deltaMap PagedFileDeltaMap, incrementFromLsn *LSN, files internal.BundleFiles,
	options TarBallFilePackerOptions) *TarBallFilePackerImpl {
	return &TarBallFilePackerImpl{
		deltaMap:         deltaMap,
		incrementFromLsn: incrementFromLsn,
		files:            files,
		options:          options,
	}
}

func (p *TarBallFilePackerImpl) getDeltaBitmapFor(filePath string) (*roaring.Bitmap, error) {
	if p.deltaMap == nil {
		return nil, nil
	}
	return p.deltaMap.GetDeltaBitmapFor(filePath)
}

func (p *TarBallFilePackerImpl) UpdateDeltaMap(deltaMap PagedFileDeltaMap) {
	p.deltaMap = deltaMap
}

// TODO : unit tests
func (p *TarBallFilePackerImpl) PackFileIntoTar(cfi *internal.ComposeFileInfo, tarBall internal.TarBall) error {
	fileReadCloser, err := p.createFileReadCloser(cfi)
	if err != nil {
		switch err.(type) {
		case SkippedFileError:
			p.files.AddSkippedFile(cfi.Header, cfi.FileInfo)
			return nil
		case internal.FileNotExistError:
			// File was deleted before opening.
			// We should ignore file here as if it did not exist.
			tracelog.WarningLogger.Println(err)
			return nil
		default:
			return err
		}
	}
	errorGroup, _ := errgroup.WithContext(context.Background())

	if p.options.verifyPageChecksums {
		var secondReadCloser io.ReadCloser
		// newTeeReadCloser is used to provide the fileReadCloser to two consumers:
		// fileReadCloser is needed for PackFileTo, secondReadCloser is for the page verification
		fileReadCloser, secondReadCloser = newTeeReadCloser(fileReadCloser)
		errorGroup.Go(func() (err error) {
			corruptBlocks, err := verifyFile(cfi.Path, cfi.FileInfo, secondReadCloser, cfi.IsIncremented)
			if err != nil {
				return err
			}
			p.files.AddFileWithCorruptBlocks(cfi.Header, cfi.FileInfo, cfi.IsIncremented,
				corruptBlocks, p.options.storeAllCorruptBlocks)
			return nil
		})
	} else {
		p.files.AddFile(cfi.Header, cfi.FileInfo, cfi.IsIncremented)
	}

	errorGroup.Go(func() error {
		defer utility.LoggedClose(fileReadCloser, "")
		packedFileSize, err := internal.PackFileTo(tarBall, cfi.Header, fileReadCloser)
		if err != nil {
			return errors.Wrap(err, "PackFileIntoTar: operation failed")
		}
		if packedFileSize != cfi.Header.Size {
			return newTarSizeError(packedFileSize, cfi.Header.Size)
		}
		return nil
	})

	return errorGroup.Wait()
}

func (p *TarBallFilePackerImpl) createFileReadCloser(cfi *internal.ComposeFileInfo) (io.ReadCloser, error) {
	var fileReadCloser io.ReadCloser
	if cfi.IsIncremented {
		bitmap, err := p.getDeltaBitmapFor(cfi.Path)
		if _, ok := err.(NoBitmapFoundError); ok { // this file has changed after the start of backup, so just skip it
			return nil, newSkippedFileError(cfi.Path)
		} else if err != nil {
			return nil, errors.Wrapf(err, "PackFileIntoTar: failed to find corresponding bitmap '%s'\n", cfi.Path)
		}
		fileReadCloser, cfi.Header.Size, err = ReadIncrementalFile(cfi.Path, cfi.FileInfo.Size(), *p.incrementFromLsn, bitmap)
		if errors.Is(err, os.ErrNotExist) {
			return nil, internal.NewFileNotExistError(cfi.Path)
		}
		switch err.(type) {
		case nil:
			fileReadCloser = &ioextensions.ReadCascadeCloser{
				Reader: &io.LimitedReader{
					R: io.MultiReader(fileReadCloser, &ioextensions.ZeroReader{}),
					N: cfi.Header.Size,
				},
				Closer: fileReadCloser,
			}
		case InvalidBlockError: // fallback to full file backup
			tracelog.WarningLogger.Printf("failed to read file '%s' as incremented\n", cfi.Header.Name)
			cfi.IsIncremented = false
			fileReadCloser, err = internal.StartReadingFile(cfi.Header, cfi.FileInfo, cfi.Path)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.Wrapf(err, "PackFileIntoTar: failed reading incremental file '%s'\n", cfi.Path)
		}
	} else {
		var err error
		fileReadCloser, err = internal.StartReadingFile(cfi.Header, cfi.FileInfo, cfi.Path)
		if err != nil {
			return nil, err
		}
	}
	return fileReadCloser, nil
}

func verifyFile(path string, fileInfo os.FileInfo, fileReader io.Reader, isIncremented bool) ([]uint32, error) {
	if !isPagedFile(fileInfo, path) {
		_, err := io.Copy(io.Discard, fileReader)
		return nil, err
	}

	if isIncremented {
		return VerifyPagedFileIncrement(path, fileInfo, fileReader)
	}
	return VerifyPagedFileBase(path, fileInfo, fileReader)
}

// TeeReadCloser creates two io.ReadClosers from one
func newTeeReadCloser(readCloser io.ReadCloser) (io.ReadCloser, io.ReadCloser) {
	pipeReader, pipeWriter := io.Pipe()

	// teeReader is used to provide the readCloser to two consumers
	teeReader := io.TeeReader(readCloser, pipeWriter)
	// MultiCloser closes both pipeWriter and readCloser on Close() call
	closer := ioextensions.NewMultiCloser([]io.Closer{readCloser, pipeWriter})
	return &ioextensions.ReadCascadeCloser{Reader: teeReader, Closer: closer}, pipeReader
}
