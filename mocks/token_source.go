package mocks

import "github.com/Bplotka/oidc"
import "github.com/stretchr/testify/mock"

// TokenSource is an autogenerated mock type for the TokenSource type
type TokenSource struct {
	mock.Mock
}

// OIDCToken provides a mock function with given fields:
func (_m *TokenSource) OIDCToken() (*oidc.Token, error) {
	ret := _m.Called()

	var r0 *oidc.Token
	if rf, ok := ret.Get(0).(func() *oidc.Token); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*oidc.Token)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Verifier provides a mock function with given fields:
func (_m *TokenSource) Verifier() oidc.Verifier {
	ret := _m.Called()

	var r0 oidc.Verifier
	if rf, ok := ret.Get(0).(func() oidc.Verifier); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(oidc.Verifier)
		}
	}

	return r0
}
