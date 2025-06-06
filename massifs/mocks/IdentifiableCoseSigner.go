// Code generated by mockery v2.52.2. DO NOT EDIT.

package mocks

import (
	context "context"
	ecdsa "crypto/ecdsa"

	cose "github.com/veraison/go-cose"

	io "io"

	mock "github.com/stretchr/testify/mock"
)

// IdentifiableCoseSigner is an autogenerated mock type for the IdentifiableCoseSigner type
type IdentifiableCoseSigner struct {
	mock.Mock
}

// Algorithm provides a mock function with no fields
func (_m *IdentifiableCoseSigner) Algorithm() cose.Algorithm {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Algorithm")
	}

	var r0 cose.Algorithm
	if rf, ok := ret.Get(0).(func() cose.Algorithm); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(cose.Algorithm)
	}

	return r0
}

// KeyIdentifier provides a mock function with no fields
func (_m *IdentifiableCoseSigner) KeyIdentifier() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for KeyIdentifier")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// KeyLocation provides a mock function with no fields
func (_m *IdentifiableCoseSigner) KeyLocation() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for KeyLocation")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// LatestPublicKey provides a mock function with no fields
func (_m *IdentifiableCoseSigner) LatestPublicKey() (*ecdsa.PublicKey, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LatestPublicKey")
	}

	var r0 *ecdsa.PublicKey
	var r1 error
	if rf, ok := ret.Get(0).(func() (*ecdsa.PublicKey, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *ecdsa.PublicKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ecdsa.PublicKey)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublicKey provides a mock function with given fields: ctx, kid
func (_m *IdentifiableCoseSigner) PublicKey(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	ret := _m.Called(ctx, kid)

	if len(ret) == 0 {
		panic("no return value specified for PublicKey")
	}

	var r0 *ecdsa.PublicKey
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*ecdsa.PublicKey, error)); ok {
		return rf(ctx, kid)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *ecdsa.PublicKey); ok {
		r0 = rf(ctx, kid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ecdsa.PublicKey)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, kid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Sign provides a mock function with given fields: rand, content
func (_m *IdentifiableCoseSigner) Sign(rand io.Reader, content []byte) ([]byte, error) {
	ret := _m.Called(rand, content)

	if len(ret) == 0 {
		panic("no return value specified for Sign")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(io.Reader, []byte) ([]byte, error)); ok {
		return rf(rand, content)
	}
	if rf, ok := ret.Get(0).(func(io.Reader, []byte) []byte); ok {
		r0 = rf(rand, content)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(io.Reader, []byte) error); ok {
		r1 = rf(rand, content)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewIdentifiableCoseSigner creates a new instance of IdentifiableCoseSigner. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIdentifiableCoseSigner(t interface {
	mock.TestingT
	Cleanup(func())
}) *IdentifiableCoseSigner {
	mock := &IdentifiableCoseSigner{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
