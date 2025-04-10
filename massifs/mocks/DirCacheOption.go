// Code generated by mockery v2.52.2. DO NOT EDIT.

package mocks

import (
	massifs "github.com/datatrails/go-datatrails-merklelog/massifs"
	mock "github.com/stretchr/testify/mock"
)

// DirCacheOption is an autogenerated mock type for the DirCacheOption type
type DirCacheOption struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *DirCacheOption) Execute(_a0 *massifs.DirCacheOptions) {
	_m.Called(_a0)
}

// NewDirCacheOption creates a new instance of DirCacheOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDirCacheOption(t interface {
	mock.TestingT
	Cleanup(func())
}) *DirCacheOption {
	mock := &DirCacheOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
