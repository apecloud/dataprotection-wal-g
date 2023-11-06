// Code generated by MockGen. DO NOT EDIT.
// Source: /home/alexander/projects/wal-g/pkg/storages/storage/folder.go

// Package mock_storage is a generated GoMock package.
package mocks

import (
	io "io"
	reflect "reflect"

	storage "github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	gomock "github.com/golang/mock/gomock"
)

// MockFolder is a mock of Folder interface.
type MockFolder struct {
	ctrl     *gomock.Controller
	recorder *MockFolderMockRecorder
}

// MockFolderMockRecorder is the mock recorder for MockFolder.
type MockFolderMockRecorder struct {
	mock *MockFolder
}

// NewMockFolder creates a new mock instance.
func NewMockFolder(ctrl *gomock.Controller) *MockFolder {
	mock := &MockFolder{ctrl: ctrl}
	mock.recorder = &MockFolderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFolder) EXPECT() *MockFolderMockRecorder {
	return m.recorder
}

// CopyObject mocks base method.
func (m *MockFolder) CopyObject(srcPath, dstPath string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CopyObject", srcPath, dstPath)
	ret0, _ := ret[0].(error)
	return ret0
}

// CopyObject indicates an expected call of CopyObject.
func (mr *MockFolderMockRecorder) CopyObject(srcPath, dstPath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CopyObject", reflect.TypeOf((*MockFolder)(nil).CopyObject), srcPath, dstPath)
}

// DeleteObjects mocks base method.
func (m *MockFolder) DeleteObjects(objectRelativePaths []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteObjects", objectRelativePaths)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteObjects indicates an expected call of DeleteObjects.
func (mr *MockFolderMockRecorder) DeleteObjects(objectRelativePaths interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteObjects", reflect.TypeOf((*MockFolder)(nil).DeleteObjects), objectRelativePaths)
}

// Exists mocks base method.
func (m *MockFolder) Exists(objectRelativePath string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists", objectRelativePath)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exists indicates an expected call of Exists.
func (mr *MockFolderMockRecorder) Exists(objectRelativePath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockFolder)(nil).Exists), objectRelativePath)
}

// GetPath mocks base method.
func (m *MockFolder) GetPath() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPath")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetPath indicates an expected call of GetPath.
func (mr *MockFolderMockRecorder) GetPath() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPath", reflect.TypeOf((*MockFolder)(nil).GetPath))
}

// GetSubFolder mocks base method.
func (m *MockFolder) GetSubFolder(subFolderRelativePath string) storage.Folder {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSubFolder", subFolderRelativePath)
	ret0, _ := ret[0].(storage.Folder)
	return ret0
}

// GetSubFolder indicates an expected call of GetSubFolder.
func (mr *MockFolderMockRecorder) GetSubFolder(subFolderRelativePath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSubFolder", reflect.TypeOf((*MockFolder)(nil).GetSubFolder), subFolderRelativePath)
}

// ListFolder mocks base method.
func (m *MockFolder) ListFolder() ([]storage.Object, []storage.Folder, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListFolder")
	ret0, _ := ret[0].([]storage.Object)
	ret1, _ := ret[1].([]storage.Folder)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ListFolder indicates an expected call of ListFolder.
func (mr *MockFolderMockRecorder) ListFolder() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFolder", reflect.TypeOf((*MockFolder)(nil).ListFolder))
}

// PutObject mocks base method.
func (m *MockFolder) PutObject(name string, content io.Reader) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PutObject", name, content)
	ret0, _ := ret[0].(error)
	return ret0
}

// PutObject indicates an expected call of PutObject.
func (mr *MockFolderMockRecorder) PutObject(name, content interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutObject", reflect.TypeOf((*MockFolder)(nil).PutObject), name, content)
}

// ReadObject mocks base method.
func (m *MockFolder) ReadObject(objectRelativePath string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadObject", objectRelativePath)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadObject indicates an expected call of ReadObject.
func (mr *MockFolderMockRecorder) ReadObject(objectRelativePath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadObject", reflect.TypeOf((*MockFolder)(nil).ReadObject), objectRelativePath)
}
