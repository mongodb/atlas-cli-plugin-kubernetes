// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store (interfaces: DatabaseUserLister)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	store "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	admin "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

// MockDatabaseUserLister is a mock of DatabaseUserLister interface.
type MockDatabaseUserLister struct {
	ctrl     *gomock.Controller
	recorder *MockDatabaseUserListerMockRecorder
}

// MockDatabaseUserListerMockRecorder is the mock recorder for MockDatabaseUserLister.
type MockDatabaseUserListerMockRecorder struct {
	mock *MockDatabaseUserLister
}

// NewMockDatabaseUserLister creates a new mock instance.
func NewMockDatabaseUserLister(ctrl *gomock.Controller) *MockDatabaseUserLister {
	mock := &MockDatabaseUserLister{ctrl: ctrl}
	mock.recorder = &MockDatabaseUserListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDatabaseUserLister) EXPECT() *MockDatabaseUserListerMockRecorder {
	return m.recorder
}

// DatabaseUsers mocks base method.
func (m *MockDatabaseUserLister) DatabaseUsers(arg0 string, arg1 *store.ListOptions) (*admin.PaginatedApiAtlasDatabaseUser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DatabaseUsers", arg0, arg1)
	ret0, _ := ret[0].(*admin.PaginatedApiAtlasDatabaseUser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DatabaseUsers indicates an expected call of DatabaseUsers.
func (mr *MockDatabaseUserListerMockRecorder) DatabaseUsers(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DatabaseUsers", reflect.TypeOf((*MockDatabaseUserLister)(nil).DatabaseUsers), arg0, arg1)
}
