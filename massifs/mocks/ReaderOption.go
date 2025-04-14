// Code generated by mockery v2.52.2. DO NOT EDIT.

package mocks

import (
	massifs "github.com/datatrails/go-datatrails-merklelog/massifs"
	mock "github.com/stretchr/testify/mock"
)

// ReaderOption is an autogenerated mock type for the ReaderOption type
type ReaderOption struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *ReaderOption) Execute(_a0 *massifs.ReaderOptions) {
	_m.Called(_a0)
}

// NewReaderOption creates a new instance of ReaderOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReaderOption(t interface {
	mock.TestingT
	Cleanup(func())
}) *ReaderOption {
	mock := &ReaderOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
