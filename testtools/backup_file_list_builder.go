package testtools

import (
	"time"

	"github.com/apecloud/dataprotection-wal-g/internal"
)

const (
	SimplePath      = "/simple"
	SkippedPath     = "/skipped"
	IncrementedPath = "/incremented"
)

var SimpleDescription = *internal.NewBackupFileDescription(false, false, time.Time{})
var SkippedDescription = *internal.NewBackupFileDescription(false, true, time.Time{})
var IncrementedDescription = *internal.NewBackupFileDescription(true, false, time.Time{})

type BackupFileListBuilder struct {
	fileList internal.BackupFileList
}

func NewBackupFileListBuilder() BackupFileListBuilder {
	return BackupFileListBuilder{internal.BackupFileList{}}
}

func (listBuilder BackupFileListBuilder) WithSimple() BackupFileListBuilder {
	listBuilder.fileList[SimplePath] = SimpleDescription
	return listBuilder
}

func (listBuilder BackupFileListBuilder) WithSkipped() BackupFileListBuilder {
	listBuilder.fileList[SkippedPath] = SkippedDescription
	return listBuilder
}

func (listBuilder BackupFileListBuilder) WithIncremented() BackupFileListBuilder {
	listBuilder.fileList[IncrementedPath] = IncrementedDescription
	return listBuilder
}

func (listBuilder BackupFileListBuilder) Build() internal.BackupFileList {
	return listBuilder.fileList
}
