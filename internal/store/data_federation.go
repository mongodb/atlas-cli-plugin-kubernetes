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

// This code was autogenerated at 2023-06-21T10:33:04+01:00. Note: Manual updates are allowed, but may be overwritten.

package store

import (
	"go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_data_federation.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store DataFederationLister,DataFederationDescriber,DataFederationStore

type DataFederationStore interface {
	DataFederationLister
	DataFederationDescriber
}

type DataFederationLister interface {
	DataFederationList(string) ([]admin.DataLakeTenant, error)
}

type DataFederationDescriber interface {
	DataFederation(string, string) (*admin.DataLakeTenant, error)
}

// DataFederationList encapsulates the logic to manage different cloud providers.
func (s *Store) DataFederationList(projectID string) ([]admin.DataLakeTenant, error) {
	req := s.clientv2.DataFederationApi.ListFederatedDatabases(s.ctx, projectID)
	result, _, err := req.Execute()
	return result, err
}

// DataFederation encapsulates the logic to manage different cloud providers.
func (s *Store) DataFederation(projectID, id string) (*admin.DataLakeTenant, error) {
	result, _, err := s.clientv2.DataFederationApi.GetFederatedDatabase(s.ctx, projectID, id).Execute()
	return result, err
}
