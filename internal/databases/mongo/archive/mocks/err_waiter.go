// Code generated by Yandex patched mockery v1.1.0. DO NOT EDIT.

package archivemocks

import mock "github.com/stretchr/testify/mock"

// ErrWaiter is an autogenerated mock type for the ErrWaiter type
type ErrWaiter struct {
	mock.Mock
}

// Wait provides a mock function with given fields:
func (_m *ErrWaiter) Wait() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
