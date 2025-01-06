// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store (interfaces: PrivateEndpointLister,InterfaceEndpointDescriber)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	admin "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

// MockPrivateEndpointLister is a mock of PrivateEndpointLister interface.
type MockPrivateEndpointLister struct {
	ctrl     *gomock.Controller
	recorder *MockPrivateEndpointListerMockRecorder
}

// MockPrivateEndpointListerMockRecorder is the mock recorder for MockPrivateEndpointLister.
type MockPrivateEndpointListerMockRecorder struct {
	mock *MockPrivateEndpointLister
}

// NewMockPrivateEndpointLister creates a new mock instance.
func NewMockPrivateEndpointLister(ctrl *gomock.Controller) *MockPrivateEndpointLister {
	mock := &MockPrivateEndpointLister{ctrl: ctrl}
	mock.recorder = &MockPrivateEndpointListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPrivateEndpointLister) EXPECT() *MockPrivateEndpointListerMockRecorder {
	return m.recorder
}

// PrivateEndpoints mocks base method.
func (m *MockPrivateEndpointLister) PrivateEndpoints(arg0, arg1 string) ([]admin.EndpointService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PrivateEndpoints", arg0, arg1)
	ret0, _ := ret[0].([]admin.EndpointService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PrivateEndpoints indicates an expected call of PrivateEndpoints.
func (mr *MockPrivateEndpointListerMockRecorder) PrivateEndpoints(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrivateEndpoints", reflect.TypeOf((*MockPrivateEndpointLister)(nil).PrivateEndpoints), arg0, arg1)
}

// MockInterfaceEndpointDescriber is a mock of InterfaceEndpointDescriber interface.
type MockInterfaceEndpointDescriber struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceEndpointDescriberMockRecorder
}

// MockInterfaceEndpointDescriberMockRecorder is the mock recorder for MockInterfaceEndpointDescriber.
type MockInterfaceEndpointDescriberMockRecorder struct {
	mock *MockInterfaceEndpointDescriber
}

// NewMockInterfaceEndpointDescriber creates a new mock instance.
func NewMockInterfaceEndpointDescriber(ctrl *gomock.Controller) *MockInterfaceEndpointDescriber {
	mock := &MockInterfaceEndpointDescriber{ctrl: ctrl}
	mock.recorder = &MockInterfaceEndpointDescriberMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterfaceEndpointDescriber) EXPECT() *MockInterfaceEndpointDescriberMockRecorder {
	return m.recorder
}

// InterfaceEndpoint mocks base method.
func (m *MockInterfaceEndpointDescriber) InterfaceEndpoint(arg0, arg1, arg2, arg3 string) (*admin.PrivateLinkEndpoint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InterfaceEndpoint", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*admin.PrivateLinkEndpoint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InterfaceEndpoint indicates an expected call of InterfaceEndpoint.
func (mr *MockInterfaceEndpointDescriberMockRecorder) InterfaceEndpoint(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InterfaceEndpoint", reflect.TypeOf((*MockInterfaceEndpointDescriber)(nil).InterfaceEndpoint), arg0, arg1, arg2, arg3)
}