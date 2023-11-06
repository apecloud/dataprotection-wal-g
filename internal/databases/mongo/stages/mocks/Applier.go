// Code generated by Yandex patched mockery v1.1.0. DO NOT EDIT.

package stagesmocks

import (
	context "context"

	models "github.com/apecloud/dataprotection-wal-g/internal/databases/mongo/models"
	mock "github.com/stretchr/testify/mock"
)

// Applier is an autogenerated mock type for the Applier type
type Applier struct {
	mock.Mock
}

// Apply provides a mock function with given fields: _a0, _a1
func (_m *Applier) Apply(_a0 context.Context, _a1 chan *models.Oplog) (chan error, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 1 {
		rf, ok := ret.Get(0).(func(context.Context, chan *models.Oplog) (chan error, error))
		if ok {
			return rf(_a0, _a1)
		}
	}

	var r0 chan error
	if rf, ok := ret.Get(0).(func(context.Context, chan *models.Oplog) chan error); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan error)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, chan *models.Oplog) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
