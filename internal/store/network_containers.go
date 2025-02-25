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
	"fmt"

	akov2provider "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/provider"
	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

const (
	MaxItems = 500
)

var (
	supportedCloudProviders = []akov2provider.ProviderName{
		akov2provider.ProviderAWS,
		akov2provider.ProviderAzure,
		akov2provider.ProviderGCP,
	}
)

//go:generate mockgen -destination=../mocks/mock_network_connections.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store NetworkContainerLister

type NetworkContainerLister interface {
	NetworkContainers(projectID string) ([]atlasv2.CloudProviderContainer, error)
}

// NetworkContainers encapsulates the logic to list all containers from all supported cloud providers
func (s *Store) NetworkContainers(projectID string) ([]atlasv2.CloudProviderContainer, error) {
	allResults := []atlasv2.CloudProviderContainer{}
	for _, provider := range supportedCloudProviders {
		results, err := s.networkContainersFor(projectID, provider)
		if err != nil {
			return nil, fmt.Errorf("error getting network containers for %s: %w", provider, err)
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func (s *Store) networkContainersFor(projectID string, provider akov2provider.ProviderName) ([]atlasv2.CloudProviderContainer, error) {
	allPages := []atlasv2.CloudProviderContainer{}
	pageNum := 0
	itemsPerPage := MaxItems
	for {
		result, _, err := s.clientv2.NetworkPeeringApi.ListPeeringContainerByCloudProvider(s.ctx, projectID).
			ItemsPerPage(itemsPerPage).
			PageNum(pageNum).
			ProviderName(string(provider)).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list network containers: %w", err)
		}
		allPages = append(allPages, result.GetResults()...)
		if len(result.GetResults()) < itemsPerPage {
			break
		}
		pageNum += 1
	}
	return allPages, nil
}
