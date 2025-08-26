// Copyright 2024 MongoDB Inc
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

	atlasv2 "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

//go:generate mockgen -destination=../mocks/mock_identity_providers_store.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store IdentityProviderLister,IdentityProviderDescriber

type IdentityProviderLister interface {
	IdentityProviders(opts *atlasv2.ListIdentityProvidersApiParams) ([]atlasv2.FederationIdentityProvider, error)
}

type IdentityProviderDescriber interface {
	IdentityProvider(opts *atlasv2.GetIdentityProviderApiParams) (*atlasv2.FederationIdentityProvider, error)
}

// IdentityProviders encapsulate the logic to manage different cloud providers.
func (s *Store) IdentityProviders(opts *atlasv2.ListIdentityProvidersApiParams) ([]atlasv2.FederationIdentityProvider, error) {
	return AllPages(func(pageNum, itemsPerPage int) ([]atlasv2.FederationIdentityProvider, error) {
		opts.PageNum = &pageNum
		opts.ItemsPerPage = &itemsPerPage

		page, _, err := s.clientv2.FederatedAuthenticationApi.ListIdentityProvidersWithParams(s.ctx, opts).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list identity providers: %w", err)
		}

		return page.GetResults(), nil
	})
}

// IdentityProvider encapsulate the logic to manage different cloud providers.
func (s *Store) IdentityProvider(opts *atlasv2.GetIdentityProviderApiParams) (*atlasv2.FederationIdentityProvider, error) {
	result, _, err := s.clientv2.FederatedAuthenticationApi.GetIdentityProviderWithParams(s.ctx, opts).Execute()
	return result, err
}
