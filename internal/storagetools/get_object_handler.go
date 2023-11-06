package storagetools

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/apecloud/dataprotection-wal-g/internal"
	"github.com/apecloud/dataprotection-wal-g/internal/compression"
	"github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	"github.com/apecloud/dataprotection-wal-g/utility"
	"github.com/wal-g/tracelog"
)

func HandleGetObject(objectPath, dstPath string, folder storage.Folder, decrypt, decompress bool) error {
	fileName := path.Base(objectPath)
	targetPath, err := getTargetFilePath(dstPath, fileName)
	if err != nil {
		return fmt.Errorf("determine the destination path: %v", err)
	}

	dstFile, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0640)
	if err != nil {
		return fmt.Errorf("open the destination file: %v", err)
	}

	err = downloadObject(objectPath, folder, dstFile, decrypt, decompress)
	dstFile.Close()
	if err != nil {
		os.Remove(targetPath)
		if err != nil {
			return fmt.Errorf("download the file: %v", err)
		}
	}

	return nil
}

func getTargetFilePath(dstPath string, fileName string) (string, error) {
	info, err := os.Stat(dstPath)
	if errors.Is(err, os.ErrNotExist) {
		return dstPath, nil
	}

	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return path.Join(dstPath, fileName), nil
	}

	return dstPath, nil
}

func downloadObject(objectPath string, folder storage.Folder, fileWriter io.Writer, decrypt, decompress bool) error {
	objReadCloser, err := folder.ReadObject(objectPath)
	if err != nil {
		return err
	}
	defer objReadCloser.Close()
	var objReader io.Reader = objReadCloser

	if decrypt {
		objReader, err = internal.DecryptBytes(objReader)
		if err != nil {
			return err
		}
	}

	if decompress {
		fileName := path.Base(objectPath)
		fileExt := path.Ext(fileName)
		decompressor := compression.FindDecompressor(fileExt)
		if decompressor == nil {
			tracelog.WarningLogger.Printf(
				"decompressor for extension '%s' was not found (supported methods: %v), will download uncompressed",
				fileExt, compression.CompressingAlgorithms)
		} else {
			decrypterObjReadCloser, err := decompressor.Decompress(objReader)
			if err != nil {
				return err
			}
			defer decrypterObjReadCloser.Close()
			objReader = decrypterObjReadCloser
		}
	}

	_, err = utility.FastCopy(fileWriter, objReader)
	return err
}
