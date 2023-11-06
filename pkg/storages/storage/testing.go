package storage

import (
	"bytes"
	"io"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// nolint: funlen
func RunFolderTest(storageFolder Folder, t *testing.T) {
	sub1 := storageFolder.GetSubFolder("Sub1")

	token := make([]byte, 1024*1024) //Send 1 Mb
	rand.Read(token)

	err := storageFolder.PutObject("file0", bytes.NewBuffer(token))
	assert.NoError(t, err)

	readCloser, err := storageFolder.ReadObject("file0")
	assert.NoError(t, err)
	all, err := io.ReadAll(readCloser)
	assert.NoError(t, err)
	assert.Equal(t, token, all)

	err = sub1.PutObject("file1", strings.NewReader("data1"))
	assert.NoError(t, err)

	b, err := storageFolder.Exists("file0")
	assert.NoError(t, err)
	assert.True(t, b)
	b, err = sub1.Exists("file1")
	assert.NoError(t, err)
	assert.True(t, b)

	objects, subFolders, err := storageFolder.ListFolder()
	assert.NoError(t, err)
	t.Log(subFolders[0].GetPath())
	assert.Equal(t, "file0", objects[0].GetName())
	assert.True(t, strings.HasSuffix(subFolders[0].GetPath(), "Sub1/") ||
		strings.HasSuffix(subFolders[0].GetPath(), "Sub1"))

	sublist, subFolders, err := sub1.ListFolder()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(subFolders))
	assert.Equal(t, 1, len(sublist))
	assert.Equal(t, sublist[0].GetName(), "file1")

	data, err := sub1.ReadObject("file1")
	assert.NoError(t, err)
	data0Str, err := io.ReadAll(data)
	assert.NoError(t, err)
	assert.Equal(t, "data1", string(data0Str))
	err = data.Close()
	assert.NoError(t, err)

	err = storageFolder.CopyObject("Sub1/file1", "Sub2/file2")
	assert.NoError(t, err)
	sublist, subFolders, err = storageFolder.ListFolder()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(subFolders))
	assert.Equal(t, 1, len(sublist))
	sublist, subFolders, err = sub1.ListFolder()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(subFolders))
	assert.Equal(t, 1, len(sublist))
	sub2 := storageFolder.GetSubFolder("Sub2")
	sublist, subFolders, err = sub2.ListFolder()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(subFolders))
	assert.Equal(t, 1, len(sublist))

	data, err = sub2.ReadObject("file2")
	assert.NoError(t, err)
	data0Str, err = io.ReadAll(data)
	assert.NoError(t, err)
	assert.Equal(t, "data1", string(data0Str))
	err = data.Close()
	assert.NoError(t, err)

	err = sub1.DeleteObjects([]string{"file1"})
	assert.NoError(t, err)
	err = storageFolder.DeleteObjects([]string{"Sub1"})
	assert.NoError(t, err)
	err = storageFolder.DeleteObjects([]string{"file0"})
	assert.NoError(t, err)

	b, err = storageFolder.Exists("file0")
	assert.NoError(t, err)
	assert.False(t, b)
	b, err = sub1.Exists("file1")
	assert.NoError(t, err)
	assert.False(t, b)

	_, err = sub1.ReadObject("Tumba Yumba")
	assert.Error(t, err.(ObjectNotFoundError))
}
