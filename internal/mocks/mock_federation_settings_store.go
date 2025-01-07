// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store (interfaces: FederationSettingsDescriber)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	admin "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

// MockFederationSettingsDescriber is a mock of FederationSettingsDescriber interface.
type MockFederationSettingsDescriber struct {
	ctrl     *gomock.Controller
	recorder *MockFederationSettingsDescriberMockRecorder
}

// MockFederationSettingsDescriberMockRecorder is the mock recorder for MockFederationSettingsDescriber.
type MockFederationSettingsDescriberMockRecorder struct {
	mock *MockFederationSettingsDescriber
}

// NewMockFederationSettingsDescriber creates a new mock instance.
func NewMockFederationSettingsDescriber(ctrl *gomock.Controller) *MockFederationSettingsDescriber {
	mock := &MockFederationSettingsDescriber{ctrl: ctrl}
	mock.recorder = &MockFederationSettingsDescriberMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFederationSettingsDescriber) EXPECT() *MockFederationSettingsDescriberMockRecorder {
	return m.recorder
}

// FederationSetting mocks base method.
func (m *MockFederationSettingsDescriber) FederationSetting(arg0 *admin.GetFederationSettingsApiParams) (*admin.OrgFederationSettings, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FederationSetting", arg0)
	ret0, _ := ret[0].(*admin.OrgFederationSettings)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FederationSetting indicates an expected call of FederationSetting.
func (mr *MockFederationSettingsDescriberMockRecorder) FederationSetting(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FederationSetting", reflect.TypeOf((*MockFederationSettingsDescriber)(nil).FederationSetting), arg0)
}
