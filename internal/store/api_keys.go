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
	atlasv2 "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

//go:generate mockgen -destination=../mocks/mock_api_keys.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store ProjectAPIKeyCreator,OrganizationAPIKeyCreator,ProjectAPIKeyAssigner

type ProjectAPIKeyCreator interface {
	CreateProjectAPIKey(string, *atlasv2.CreateAtlasProjectApiKey) (*atlasv2.ApiKeyUserDetails, error)
}
type ProjectAPIKeyAssigner interface {
	AssignProjectAPIKey(string, string, *atlasv2.UpdateAtlasProjectApiKey) error
}

type OrganizationAPIKeyCreator interface {
	CreateOrganizationAPIKey(string, *atlasv2.CreateAtlasOrganizationApiKey) (*atlasv2.ApiKeyUserDetails, error)
}

// CreateOrganizationAPIKey encapsulates the logic to manage different cloud providers.
func (s *Store) CreateOrganizationAPIKey(orgID string, input *atlasv2.CreateAtlasOrganizationApiKey) (*atlasv2.ApiKeyUserDetails, error) {
	result, _, err := s.clientv2.ProgrammaticAPIKeysApi.CreateApiKey(s.ctx, orgID, input).Execute()
	return result, err
}

// CreateProjectAPIKey creates an API Keys for a project.
func (s *Store) CreateProjectAPIKey(projectID string, apiKeyInput *atlasv2.CreateAtlasProjectApiKey) (*atlasv2.ApiKeyUserDetails, error) {
	result, _, err := s.clientv2.ProgrammaticAPIKeysApi.CreateProjectApiKey(s.ctx, projectID, apiKeyInput).Execute()
	return result, err
}

// AssignProjectAPIKey encapsulates the logic to manage different cloud providers.
func (s *Store) AssignProjectAPIKey(projectID, apiKeyID string, input *atlasv2.UpdateAtlasProjectApiKey) error {
	_, _, err := s.clientv2.ProgrammaticAPIKeysApi.UpdateApiKeyRoles(s.ctx, projectID, apiKeyID, input).Execute()
	return err
}
