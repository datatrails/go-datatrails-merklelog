// Code generated by mockery v2.42.2. DO NOT EDIT.

package mocks

import (
	context "context"

	azblob "github.com/datatrails/go-datatrails-common/azblob"

	mock "github.com/stretchr/testify/mock"
)

// LogBlobReader is an autogenerated mock type for the LogBlobReader type
type LogBlobReader struct {
	mock.Mock
}

// FilteredList provides a mock function with given fields: ctx, tagsFilter, opts
func (_m *LogBlobReader) FilteredList(ctx context.Context, tagsFilter string, opts ...azblob.Option) (*azblob.FilterResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, tagsFilter)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for FilteredList")
	}

	var r0 *azblob.FilterResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, ...azblob.Option) (*azblob.FilterResponse, error)); ok {
		return rf(ctx, tagsFilter, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, ...azblob.Option) *azblob.FilterResponse); ok {
		r0 = rf(ctx, tagsFilter, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*azblob.FilterResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, ...azblob.Option) error); ok {
		r1 = rf(ctx, tagsFilter, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: ctx, opts
func (_m *LogBlobReader) List(ctx context.Context, opts ...azblob.Option) (*azblob.ListerResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 *azblob.ListerResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ...azblob.Option) (*azblob.ListerResponse, error)); ok {
		return rf(ctx, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ...azblob.Option) *azblob.ListerResponse); ok {
		r0 = rf(ctx, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*azblob.ListerResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ...azblob.Option) error); ok {
		r1 = rf(ctx, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Reader provides a mock function with given fields: ctx, identity, opts
func (_m *LogBlobReader) Reader(ctx context.Context, identity string, opts ...azblob.Option) (*azblob.ReaderResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, identity)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Reader")
	}

	var r0 *azblob.ReaderResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, ...azblob.Option) (*azblob.ReaderResponse, error)); ok {
		return rf(ctx, identity, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, ...azblob.Option) *azblob.ReaderResponse); ok {
		r0 = rf(ctx, identity, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*azblob.ReaderResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, ...azblob.Option) error); ok {
		r1 = rf(ctx, identity, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewLogBlobReader creates a new instance of LogBlobReader. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewLogBlobReader(t interface {
	mock.TestingT
	Cleanup(func())
}) *LogBlobReader {
	mock := &LogBlobReader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
