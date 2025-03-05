// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store (interfaces: PeeringConnectionLister)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	admin "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

// MockPeeringConnectionLister is a mock of PeeringConnectionLister interface.
type MockPeeringConnectionLister struct {
	ctrl     *gomock.Controller
	recorder *MockPeeringConnectionListerMockRecorder
}

// MockPeeringConnectionListerMockRecorder is the mock recorder for MockPeeringConnectionLister.
type MockPeeringConnectionListerMockRecorder struct {
	mock *MockPeeringConnectionLister
}

// NewMockPeeringConnectionLister creates a new mock instance.
func NewMockPeeringConnectionLister(ctrl *gomock.Controller) *MockPeeringConnectionLister {
	mock := &MockPeeringConnectionLister{ctrl: ctrl}
	mock.recorder = &MockPeeringConnectionListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPeeringConnectionLister) EXPECT() *MockPeeringConnectionListerMockRecorder {
	return m.recorder
}

// PeeringConnections mocks base method.
func (m *MockPeeringConnectionLister) PeeringConnections(arg0 string) ([]admin.BaseNetworkPeeringConnectionSettings, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PeeringConnections", arg0)
	ret0, _ := ret[0].([]admin.BaseNetworkPeeringConnectionSettings)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PeeringConnections indicates an expected call of PeeringConnections.
func (mr *MockPeeringConnectionListerMockRecorder) PeeringConnections(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PeeringConnections", reflect.TypeOf((*MockPeeringConnectionLister)(nil).PeeringConnections), arg0)
}
