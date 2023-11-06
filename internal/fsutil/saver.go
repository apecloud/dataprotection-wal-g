package fsutil

import (
	"io"
)

type Saver interface {
	Save(writer io.Writer) error
}

func SaveToDataFolder(saver Saver, filename string, dataFolder DataFolder) error {
	file, err := dataFolder.OpenWriteOnlyFile(filename)
	if err != nil {
		return err
	}
	err = saver.Save(file)
	if err != nil {
		return err
	}
	return file.Close()
}
