// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/crds (interfaces: AtlasOperatorCRDProvider)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// MockAtlasOperatorCRDProvider is a mock of AtlasOperatorCRDProvider interface.
type MockAtlasOperatorCRDProvider struct {
	ctrl     *gomock.Controller
	recorder *MockAtlasOperatorCRDProviderMockRecorder
}

// MockAtlasOperatorCRDProviderMockRecorder is the mock recorder for MockAtlasOperatorCRDProvider.
type MockAtlasOperatorCRDProviderMockRecorder struct {
	mock *MockAtlasOperatorCRDProvider
}

// NewMockAtlasOperatorCRDProvider creates a new mock instance.
func NewMockAtlasOperatorCRDProvider(ctrl *gomock.Controller) *MockAtlasOperatorCRDProvider {
	mock := &MockAtlasOperatorCRDProvider{ctrl: ctrl}
	mock.recorder = &MockAtlasOperatorCRDProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAtlasOperatorCRDProvider) EXPECT() *MockAtlasOperatorCRDProviderMockRecorder {
	return m.recorder
}

// GetAtlasOperatorResource mocks base method.
func (m *MockAtlasOperatorCRDProvider) GetAtlasOperatorResource(arg0, arg1 string) (*v1.CustomResourceDefinition, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAtlasOperatorResource", arg0, arg1)
	ret0, _ := ret[0].(*v1.CustomResourceDefinition)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAtlasOperatorResource indicates an expected call of GetAtlasOperatorResource.
func (mr *MockAtlasOperatorCRDProviderMockRecorder) GetAtlasOperatorResource(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAtlasOperatorResource", reflect.TypeOf((*MockAtlasOperatorCRDProvider)(nil).GetAtlasOperatorResource), arg0, arg1)
}