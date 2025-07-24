// Copyright 2023 MongoDB Inc
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

	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_streams.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store StreamsLister,StreamsConnectionLister

type StreamsLister interface {
	ProjectStreams(string) ([]atlasv2.StreamsTenant, error)
}

type StreamsConnectionLister interface {
	StreamsConnections(string, string) (*atlasv2.PaginatedApiStreamsConnection, error)
}

func (s *Store) ProjectStreams(projectID string) ([]atlasv2.StreamsTenant, error) {
	return AllPages(func(pageNum, itemsPerPage int) ([]atlasv2.StreamsTenant, error) {
		page, _, err := s.clientv2.StreamsApi.ListStreamInstances(s.ctx, projectID).
			PageNum(pageNum).
			ItemsPerPage(itemsPerPage).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list stream instances: %w", err)
		}
		return page.GetResults(), nil
	})
}

// StreamsConnections encapsulates the logic to manage different cloud providers.
func (s *Store) StreamsConnections(projectID, tenantName string) (*atlasv2.PaginatedApiStreamsConnection, error) {
	connections, _, err := s.clientv2.StreamsApi.ListStreamConnections(s.ctx, projectID, tenantName).Execute()
	return connections, err
}
