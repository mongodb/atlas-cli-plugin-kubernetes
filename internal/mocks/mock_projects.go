// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store (interfaces: ProjectLister,ProjectCreator,ProjectDescriber,ProjectTeamLister,OrgProjectLister)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	store "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	admin "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

// MockProjectLister is a mock of ProjectLister interface.
type MockProjectLister struct {
	ctrl     *gomock.Controller
	recorder *MockProjectListerMockRecorder
}

// MockProjectListerMockRecorder is the mock recorder for MockProjectLister.
type MockProjectListerMockRecorder struct {
	mock *MockProjectLister
}

// NewMockProjectLister creates a new mock instance.
func NewMockProjectLister(ctrl *gomock.Controller) *MockProjectLister {
	mock := &MockProjectLister{ctrl: ctrl}
	mock.recorder = &MockProjectListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProjectLister) EXPECT() *MockProjectListerMockRecorder {
	return m.recorder
}

// Projects mocks base method.
func (m *MockProjectLister) Projects(arg0 *store.ListOptions) (*admin.PaginatedAtlasGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Projects", arg0)
	ret0, _ := ret[0].(*admin.PaginatedAtlasGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Projects indicates an expected call of Projects.
func (mr *MockProjectListerMockRecorder) Projects(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Projects", reflect.TypeOf((*MockProjectLister)(nil).Projects), arg0)
}

// MockProjectCreator is a mock of ProjectCreator interface.
type MockProjectCreator struct {
	ctrl     *gomock.Controller
	recorder *MockProjectCreatorMockRecorder
}

// MockProjectCreatorMockRecorder is the mock recorder for MockProjectCreator.
type MockProjectCreatorMockRecorder struct {
	mock *MockProjectCreator
}

// NewMockProjectCreator creates a new mock instance.
func NewMockProjectCreator(ctrl *gomock.Controller) *MockProjectCreator {
	mock := &MockProjectCreator{ctrl: ctrl}
	mock.recorder = &MockProjectCreatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProjectCreator) EXPECT() *MockProjectCreatorMockRecorder {
	return m.recorder
}

// CreateProject mocks base method.
func (m *MockProjectCreator) CreateProject(arg0 *admin.CreateProjectApiParams) (*admin.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateProject", arg0)
	ret0, _ := ret[0].(*admin.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateProject indicates an expected call of CreateProject.
func (mr *MockProjectCreatorMockRecorder) CreateProject(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateProject", reflect.TypeOf((*MockProjectCreator)(nil).CreateProject), arg0)
}

// MockProjectDescriber is a mock of ProjectDescriber interface.
type MockProjectDescriber struct {
	ctrl     *gomock.Controller
	recorder *MockProjectDescriberMockRecorder
}

// MockProjectDescriberMockRecorder is the mock recorder for MockProjectDescriber.
type MockProjectDescriberMockRecorder struct {
	mock *MockProjectDescriber
}

// NewMockProjectDescriber creates a new mock instance.
func NewMockProjectDescriber(ctrl *gomock.Controller) *MockProjectDescriber {
	mock := &MockProjectDescriber{ctrl: ctrl}
	mock.recorder = &MockProjectDescriberMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProjectDescriber) EXPECT() *MockProjectDescriberMockRecorder {
	return m.recorder
}

// Project mocks base method.
func (m *MockProjectDescriber) Project(arg0 string) (*admin.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Project", arg0)
	ret0, _ := ret[0].(*admin.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Project indicates an expected call of Project.
func (mr *MockProjectDescriberMockRecorder) Project(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Project", reflect.TypeOf((*MockProjectDescriber)(nil).Project), arg0)
}

// ProjectByName mocks base method.
func (m *MockProjectDescriber) ProjectByName(arg0 string) (*admin.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProjectByName", arg0)
	ret0, _ := ret[0].(*admin.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ProjectByName indicates an expected call of ProjectByName.
func (mr *MockProjectDescriberMockRecorder) ProjectByName(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProjectByName", reflect.TypeOf((*MockProjectDescriber)(nil).ProjectByName), arg0)
}

// MockProjectTeamLister is a mock of ProjectTeamLister interface.
type MockProjectTeamLister struct {
	ctrl     *gomock.Controller
	recorder *MockProjectTeamListerMockRecorder
}

// MockProjectTeamListerMockRecorder is the mock recorder for MockProjectTeamLister.
type MockProjectTeamListerMockRecorder struct {
	mock *MockProjectTeamLister
}

// NewMockProjectTeamLister creates a new mock instance.
func NewMockProjectTeamLister(ctrl *gomock.Controller) *MockProjectTeamLister {
	mock := &MockProjectTeamLister{ctrl: ctrl}
	mock.recorder = &MockProjectTeamListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProjectTeamLister) EXPECT() *MockProjectTeamListerMockRecorder {
	return m.recorder
}

// ProjectTeams mocks base method.
func (m *MockProjectTeamLister) ProjectTeams(arg0 string, arg1 *store.ListOptions) (*admin.PaginatedTeamRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProjectTeams", arg0, arg1)
	ret0, _ := ret[0].(*admin.PaginatedTeamRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ProjectTeams indicates an expected call of ProjectTeams.
func (mr *MockProjectTeamListerMockRecorder) ProjectTeams(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProjectTeams", reflect.TypeOf((*MockProjectTeamLister)(nil).ProjectTeams), arg0, arg1)
}

// MockOrgProjectLister is a mock of OrgProjectLister interface.
type MockOrgProjectLister struct {
	ctrl     *gomock.Controller
	recorder *MockOrgProjectListerMockRecorder
}

// MockOrgProjectListerMockRecorder is the mock recorder for MockOrgProjectLister.
type MockOrgProjectListerMockRecorder struct {
	mock *MockOrgProjectLister
}

// NewMockOrgProjectLister creates a new mock instance.
func NewMockOrgProjectLister(ctrl *gomock.Controller) *MockOrgProjectLister {
	mock := &MockOrgProjectLister{ctrl: ctrl}
	mock.recorder = &MockOrgProjectListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOrgProjectLister) EXPECT() *MockOrgProjectListerMockRecorder {
	return m.recorder
}

// GetOrgProjects mocks base method.
func (m *MockOrgProjectLister) GetOrgProjects(arg0 string, arg1 *store.ListOptions) (*admin.PaginatedAtlasGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOrgProjects", arg0, arg1)
	ret0, _ := ret[0].(*admin.PaginatedAtlasGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOrgProjects indicates an expected call of GetOrgProjects.
func (mr *MockOrgProjectListerMockRecorder) GetOrgProjects(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOrgProjects", reflect.TypeOf((*MockOrgProjectLister)(nil).GetOrgProjects), arg0, arg1)
}

// Projects mocks base method.
func (m *MockOrgProjectLister) Projects(arg0 *store.ListOptions) (*admin.PaginatedAtlasGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Projects", arg0)
	ret0, _ := ret[0].(*admin.PaginatedAtlasGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Projects indicates an expected call of Projects.
func (mr *MockOrgProjectListerMockRecorder) Projects(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Projects", reflect.TypeOf((*MockOrgProjectLister)(nil).Projects), arg0)
}