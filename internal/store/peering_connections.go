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
	"fmt"

	akov2provider "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/provider"
	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_peering_connections.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store PeeringConnectionLister

type PeeringConnectionLister interface {
	PeeringConnections(string) ([]atlasv2.BaseNetworkPeeringConnectionSettings, error)
}

// PeeringConnections encapsulates the logic to list all peerings from all cloud providers
func (s *Store) PeeringConnections(projectID string) ([]atlasv2.BaseNetworkPeeringConnectionSettings, error) {
	allResults := []atlasv2.BaseNetworkPeeringConnectionSettings{}
	for _, provider := range supportedCloudProviders {
		results, err := s.peeringConnectionsFor(projectID, provider)
		if err != nil {
			return nil, fmt.Errorf("error getting network peering connections for %s: %w", provider, err)
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func (s *Store) peeringConnectionsFor(projectID string, provider akov2provider.ProviderName) ([]atlasv2.BaseNetworkPeeringConnectionSettings, error) {
	return AllPages(func(pageNum, itemsPerPage int) ([]atlasv2.BaseNetworkPeeringConnectionSettings, error) {
		page, _, err := s.clientv2.NetworkPeeringApi.ListPeeringConnections(s.ctx, projectID).
			ItemsPerPage(itemsPerPage).
			PageNum(pageNum).
			ProviderName(string(provider)).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list network peerings: %w", err)
		}
		return page.GetResults(), nil
	})
}
