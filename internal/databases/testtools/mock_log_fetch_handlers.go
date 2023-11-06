// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/stream_fetch_helper.go

// Package mock_internal is a generated GoMock package.
package mock_internal

import (
	reflect "reflect"
	time "time"

	storage "github.com/apecloud/dataprotection-wal-g/pkg/storages/storage"
	gomock "github.com/golang/mock/gomock"
)

// MockLogFetchSettings is a mock of LogFetchSettings interface
type MockLogFetchSettings struct {
	ctrl     *gomock.Controller
	recorder *MockLogFetchSettingsMockRecorder
}

// MockLogFetchSettingsMockRecorder is the mock recorder for MockLogFetchSettings
type MockLogFetchSettingsMockRecorder struct {
	mock *MockLogFetchSettings
}

// NewMockLogFetchSettings creates a new mock instance
func NewMockLogFetchSettings(ctrl *gomock.Controller) *MockLogFetchSettings {
	mock := &MockLogFetchSettings{ctrl: ctrl}
	mock.recorder = &MockLogFetchSettingsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockLogFetchSettings) EXPECT() *MockLogFetchSettingsMockRecorder {
	return m.recorder
}

// GetLogFolderPath mocks base method
func (m *MockLogFetchSettings) GetLogFolderPath() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLogFolderPath")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetLogFolderPath indicates an expected call of GetLogFolderPath
func (mr *MockLogFetchSettingsMockRecorder) GetLogFolderPath() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLogFolderPath", reflect.TypeOf((*MockLogFetchSettings)(nil).GetLogFolderPath))
}

// GetLogsFetchInterval mocks base method
func (m *MockLogFetchSettings) GetLogsFetchInterval() (time.Time, *time.Time) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLogsFetchInterval")
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(*time.Time)
	return ret0, ret1
}

// GetLogsFetchInterval indicates an expected call of GetLogsFetchInterval
func (mr *MockLogFetchSettingsMockRecorder) GetLogsFetchInterval() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLogsFetchInterval", reflect.TypeOf((*MockLogFetchSettings)(nil).GetLogsFetchInterval))
}

// MockLogFetchHandlers is a mock of LogFetchHandlers interface
type MockLogFetchHandlers struct {
	ctrl     *gomock.Controller
	recorder *MockLogFetchHandlersMockRecorder
}

// MockLogFetchHandlersMockRecorder is the mock recorder for MockLogFetchHandlers
type MockLogFetchHandlersMockRecorder struct {
	mock *MockLogFetchHandlers
}

// NewMockLogFetchHandlers creates a new mock instance
func NewMockLogFetchHandlers(ctrl *gomock.Controller) *MockLogFetchHandlers {
	mock := &MockLogFetchHandlers{ctrl: ctrl}
	mock.recorder = &MockLogFetchHandlersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockLogFetchHandlers) EXPECT() *MockLogFetchHandlersMockRecorder {
	return m.recorder
}

// FetchLog mocks base method
func (m *MockLogFetchHandlers) FetchLog(logFolder storage.Folder, logName string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchLog", logFolder, logName)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchLog indicates an expected call of FetchLog
func (mr *MockLogFetchHandlersMockRecorder) FetchLog(logFolder, logName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchLog", reflect.TypeOf((*MockLogFetchHandlers)(nil).FetchLog), logFolder, logName)
}

// HandleAbortFetch mocks base method
func (m *MockLogFetchHandlers) HandleAbortFetch(LogName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleAbortFetch", LogName)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleAbortFetch indicates an expected call of HandleAbortFetch
func (mr *MockLogFetchHandlersMockRecorder) HandleAbortFetch(LogName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleAbortFetch", reflect.TypeOf((*MockLogFetchHandlers)(nil).HandleAbortFetch), LogName)
}

// AfterFetch mocks base method
func (m *MockLogFetchHandlers) AfterFetch(logs []storage.Object) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AfterFetch", logs)
	ret0, _ := ret[0].(error)
	return ret0
}

// AfterFetch indicates an expected call of AfterFetch
func (mr *MockLogFetchHandlersMockRecorder) AfterFetch(logs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AfterFetch", reflect.TypeOf((*MockLogFetchHandlers)(nil).AfterFetch), logs)
}
