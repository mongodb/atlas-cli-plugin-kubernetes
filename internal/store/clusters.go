// Copyright 2025 MongoDB Inc
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
	atlasClustersPinned "go.mongodb.org/atlas-sdk/v20240530005/admin"
	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_clusters.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store ClusterLister,ClusterDescriber,AtlasClusterConfigurationOptionsDescriber

type ClusterLister interface {
	ProjectClusters(string, *ListOptions) (*atlasClustersPinned.PaginatedAdvancedClusterDescription, error)
	ListFlexClusters(*atlasv2.ListFlexClustersApiParams) (*atlasv2.PaginatedFlexClusters20241113, error)
}

type ClusterDescriber interface {
	AtlasCluster(string, string) (*atlasClustersPinned.AdvancedClusterDescription, error)
	FlexCluster(string, string) (*atlasv2.FlexClusterDescription20241113, error)
}

type AtlasClusterConfigurationOptionsDescriber interface {
	AtlasClusterConfigurationOptions(string, string) (*atlasClustersPinned.ClusterDescriptionProcessArgs, error)
}

// ProjectClusters encapsulate the logic to manage different cloud providers.
func (s *Store) ProjectClusters(projectID string, opts *ListOptions) (*atlasClustersPinned.PaginatedAdvancedClusterDescription, error) {
	res := s.clientClusters.ClustersApi.ListClusters(s.ctx, projectID)
	if opts != nil {
		res = res.PageNum(opts.PageNum).ItemsPerPage(fixPageSize(opts.ItemsPerPage)).IncludeCount(opts.IncludeCount)
	}
	result, _, err := res.Execute()
	return result, err
}

// AtlasCluster encapsulates the logic to manage different cloud providers.
func (s *Store) AtlasCluster(projectID, name string) (*atlasClustersPinned.AdvancedClusterDescription, error) {
	result, _, err := s.clientClusters.ClustersApi.GetCluster(s.ctx, projectID, name).Execute()
	return result, err
}

// AtlasClusterConfigurationOptions encapsulates the logic to manage different cloud providers.
func (s *Store) AtlasClusterConfigurationOptions(projectID, name string) (*atlasClustersPinned.ClusterDescriptionProcessArgs, error) {
	result, _, err := s.clientClusters.ClustersApi.GetClusterAdvancedConfiguration(s.ctx, projectID, name).Execute()
	return result, err
}
