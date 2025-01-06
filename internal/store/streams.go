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
	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_streams.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store StreamsLister,StreamsConnectionLister

type StreamsLister interface {
	ProjectStreams(*atlasv2.ListStreamInstancesApiParams) (*atlasv2.PaginatedApiStreamsTenant, error)
}

type StreamsConnectionLister interface {
	StreamsConnections(string, string) (*atlasv2.PaginatedApiStreamsConnection, error)
}

func (s *Store) ProjectStreams(opts *atlasv2.ListStreamInstancesApiParams) (*atlasv2.PaginatedApiStreamsTenant, error) {
	result, _, err := s.clientv2.StreamsApi.ListStreamInstancesWithParams(s.ctx, opts).Execute()
	return result, err
}

// StreamsConnections encapsulates the logic to manage different cloud providers.
func (s *Store) StreamsConnections(projectID, tenantName string) (*atlasv2.PaginatedApiStreamsConnection, error) {
	connections, _, err := s.clientv2.StreamsApi.ListStreamConnections(s.ctx, projectID, tenantName).Execute()
	return connections, err
}
