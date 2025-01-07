// Copyright 2021 MongoDB Inc
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
	"fmt"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/config"
	atlasClustersPinned "go.mongodb.org/atlas-sdk/v20240530005/admin"
)

//go:generate mockgen -destination=../mocks/mock_serverless_instances.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store ServerlessInstanceLister,ServerlessInstanceDescriber

type ServerlessInstanceLister interface {
	ServerlessInstances(string, *ListOptions) (*atlasClustersPinned.PaginatedServerlessInstanceDescription, error)
}

type ServerlessInstanceDescriber interface {
	GetServerlessInstance(string, string) (*atlasClustersPinned.ServerlessInstanceDescription, error)
}

// ServerlessInstances encapsulates the logic to manage different cloud providers.
func (s *Store) ServerlessInstances(projectID string, listOps *ListOptions) (*atlasClustersPinned.PaginatedServerlessInstanceDescription, error) {
	if s.service == config.CloudGovService {
		return nil, fmt.Errorf("%w: %s", errUnsupportedService, s.service)
	}
	result, _, err := s.clientClusters.ServerlessInstancesApi.ListServerlessInstances(s.ctx, projectID).
		ItemsPerPage(listOps.ItemsPerPage).
		PageNum(listOps.PageNum).
		IncludeCount(listOps.IncludeCount).
		Execute()

	return result, err
}

// GetServerlessInstance encapsulates the logic to manage different cloud providers.
func (s *Store) GetServerlessInstance(projectID, clusterName string) (*atlasClustersPinned.ServerlessInstanceDescription, error) {
	if s.service == config.CloudGovService {
		return nil, fmt.Errorf("%w: %s", errUnsupportedService, s.service)
	}
	result, _, err := s.clientClusters.ServerlessInstancesApi.GetServerlessInstance(s.ctx, projectID, clusterName).Execute()
	return result, err
}
