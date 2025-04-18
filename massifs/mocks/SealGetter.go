// Code generated by mockery v2.52.2. DO NOT EDIT.

package mocks

import (
	context "context"

	cose "github.com/datatrails/go-datatrails-common/cose"
	massifs "github.com/datatrails/go-datatrails-merklelog/massifs"

	mock "github.com/stretchr/testify/mock"
)

// SealGetter is an autogenerated mock type for the SealGetter type
type SealGetter struct {
	mock.Mock
}

// GetSignedRoot provides a mock function with given fields: ctx, tenantIdentity, massifIndex, opts
func (_m *SealGetter) GetSignedRoot(ctx context.Context, tenantIdentity string, massifIndex uint32, opts ...massifs.ReaderOption) (*cose.CoseSign1Message, massifs.MMRState, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, tenantIdentity, massifIndex)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetSignedRoot")
	}

	var r0 *cose.CoseSign1Message
	var r1 massifs.MMRState
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, uint32, ...massifs.ReaderOption) (*cose.CoseSign1Message, massifs.MMRState, error)); ok {
		return rf(ctx, tenantIdentity, massifIndex, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, uint32, ...massifs.ReaderOption) *cose.CoseSign1Message); ok {
		r0 = rf(ctx, tenantIdentity, massifIndex, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*cose.CoseSign1Message)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, uint32, ...massifs.ReaderOption) massifs.MMRState); ok {
		r1 = rf(ctx, tenantIdentity, massifIndex, opts...)
	} else {
		r1 = ret.Get(1).(massifs.MMRState)
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, uint32, ...massifs.ReaderOption) error); ok {
		r2 = rf(ctx, tenantIdentity, massifIndex, opts...)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// NewSealGetter creates a new instance of SealGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSealGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *SealGetter {
	mock := &SealGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
