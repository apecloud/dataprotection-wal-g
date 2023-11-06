package storagetools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/apecloud/dataprotection-wal-g/internal/compression"
	"github.com/apecloud/dataprotection-wal-g/internal/crypto"

	"github.com/apecloud/dataprotection-wal-g/utility"

	"github.com/apecloud/dataprotection-wal-g/internal"
)

func HandlePutObject(localPath, dstPath string, uploader internal.Uploader, overwrite, encrypt, compress bool) error {
	err := checkOverwrite(dstPath, uploader, overwrite)
	if err != nil {
		return fmt.Errorf("check file overwrite: %v", err)
	}

	fileReadCloser, err := openLocalFile(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %v", err)
	}

	defer fileReadCloser.Close()

	storageFolderPath := utility.SanitizePath(filepath.Dir(dstPath))
	if storageFolderPath != "" {
		uploader.ChangeDirectory(storageFolderPath)
	}

	fileName := utility.SanitizePath(filepath.Base(dstPath))
	err = uploadFile(fileName, fileReadCloser, uploader, encrypt, compress)
	if err != nil {
		return fmt.Errorf("upload: %v", err)
	}
	return nil
}

func checkOverwrite(dstPath string, uploader internal.Uploader, overwrite bool) error {
	fullPath := dstPath + "." + uploader.Compression().FileExtension()
	exists, err := uploader.Folder().Exists(fullPath)
	if err != nil {
		return fmt.Errorf("check object existence: %v", err)
	}
	if exists && !overwrite {
		return fmt.Errorf("object %s already exists. To overwrite it, add the -f flag", fullPath)
	}
	return nil
}

func openLocalFile(localPath string) (io.ReadCloser, error) {
	localFile, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("open the local file: %v", err)
	}

	fileInfo, err := localFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat() the local file: %v", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("provided local path (%s) points to a directory, exiting", localPath)
	}

	return localFile, nil
}

func uploadFile(name string, content io.Reader, uploader internal.Uploader, encrypt, compress bool) error {
	var crypter crypto.Crypter
	if encrypt {
		crypter = internal.ConfigureCrypter()
	}

	var compressor compression.Compressor
	if compress && uploader.Compression() != nil {
		compressor = uploader.Compression()
		name += "." + uploader.Compression().FileExtension()
	}

	uploadContents := internal.CompressAndEncrypt(content, compressor, crypter)
	return uploader.Upload(name, uploadContents)
}
