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
	atlasv2 "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

//go:generate mockgen -destination=../mocks/mock_private_endpoints.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store PrivateEndpointLister,InterfaceEndpointDescriber

type PrivateEndpointLister interface {
	PrivateEndpoints(string, string) ([]atlasv2.EndpointService, error)
}

type InterfaceEndpointDescriber interface {
	InterfaceEndpoint(projectID, cloudProvider, privateEndpointID, endpointServiceID string) (*atlasv2.PrivateLinkEndpoint, error)
}

// PrivateEndpoints encapsulates the logic to manage different cloud providers.
func (s *Store) PrivateEndpoints(projectID, provider string) ([]atlasv2.EndpointService, error) {
	result, _, err := s.clientv2.PrivateEndpointServicesApi.ListPrivateEndpointServices(s.ctx, projectID, provider).Execute()
	return result, err
}

// InterfaceEndpoint encapsulates the logic to manage different cloud providers.
func (s *Store) InterfaceEndpoint(projectID, cloudProvider, privateEndpointID, endpointServiceID string) (*atlasv2.PrivateLinkEndpoint, error) {
	result, _, err := s.clientv2.PrivateEndpointServicesApi.GetPrivateEndpoint(s.ctx, projectID, cloudProvider, privateEndpointID, endpointServiceID).Execute()
	return result, err
}
