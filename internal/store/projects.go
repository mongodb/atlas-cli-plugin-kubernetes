// Copyright 2020 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_projects.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store ProjectLister,ProjectCreator,ProjectDescriber,ProjectTeamLister,OrgProjectLister

type ProjectLister interface {
	Projects(*ListOptions) (*atlasv2.PaginatedAtlasGroup, error)
}

type OrgProjectLister interface {
	ProjectLister
	GetOrgProjects(string, *ListOptions) (*atlasv2.PaginatedAtlasGroup, error)
}

type ProjectCreator interface {
	CreateProject(*atlasv2.CreateProjectApiParams) (*atlasv2.Group, error)
}

type ProjectDescriber interface {
	Project(string) (*atlasv2.Group, error)
	ProjectByName(string) (*atlasv2.Group, error)
}

type ProjectTeamLister interface {
	ProjectTeams(string, *ListOptions) (*atlasv2.PaginatedTeamRole, error)
}

// Projects encapsulates the logic to manage different cloud providers.
func (s *Store) Projects(opts *ListOptions) (*atlasv2.PaginatedAtlasGroup, error) {
	res := s.clientv2.ProjectsApi.ListProjects(s.ctx)
	if opts != nil {
		res = res.PageNum(opts.PageNum).ItemsPerPage(fixPageSize(opts.ItemsPerPage))
	}
	result, _, err := res.Execute()
	return result, err
}

// GetOrgProjects encapsulates the logic to manage different cloud providers.
func (s *Store) GetOrgProjects(orgID string, opts *ListOptions) (*atlasv2.PaginatedAtlasGroup, error) {
	res := s.clientv2.OrganizationsApi.ListOrganizationProjects(s.ctx, orgID)
	if opts != nil {
		res = res.PageNum(opts.PageNum).ItemsPerPage(fixPageSize(opts.ItemsPerPage))
	}
	result, _, err := res.Execute()
	return result, err
}

// Project encapsulates the logic to manage different cloud providers.
func (s *Store) Project(id string) (*atlasv2.Group, error) {
	result, _, err := s.clientv2.ProjectsApi.GetProject(s.ctx, id).Execute()
	return result, err
}

func (s *Store) ProjectByName(name string) (*atlasv2.Group, error) {
	result, _, err := s.clientv2.ProjectsApi.GetProjectByName(s.ctx, name).Execute()
	return result, err
}

// CreateProject encapsulates the logic to manage different cloud providers.
func (s *Store) CreateProject(params *atlasv2.CreateProjectApiParams) (*atlasv2.Group, error) {
	result, _, err := s.clientv2.ProjectsApi.CreateProjectWithParams(s.ctx, params).Execute()
	return result, err
}

// ProjectTeams encapsulates the logic to manage different cloud providers.
func (s *Store) ProjectTeams(projectID string, opts *ListOptions) (*atlasv2.PaginatedTeamRole, error) {
	res := s.clientv2.TeamsApi.
		ListProjectTeams(s.ctx, projectID)

	if opts != nil {
		res.
			IncludeCount(opts.IncludeCount).
			PageNum(opts.PageNum).
			ItemsPerPage(fixPageSize(opts.ItemsPerPage))
	}

	result, _, err := res.Execute()
	return result, err
}

func fixPageSize(itemsPerPage int) int {
	if itemsPerPage < 1 {
		return MaxAPIPageSize
	}
	return itemsPerPage
}
