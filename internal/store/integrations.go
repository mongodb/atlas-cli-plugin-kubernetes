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

	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_integrations.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store IntegrationLister

type IntegrationLister interface {
	Integrations(string) (*atlasv2.PaginatedIntegration, error)
	AllIntegrations(string) ([]atlasv2.ThirdPartyIntegration, error)
}

// Integrations encapsulates the logic to manage different cloud providers.
func (s *Store) Integrations(projectID string) (*atlasv2.PaginatedIntegration, error) {
	resp, _, err := s.clientv2.ThirdPartyIntegrationsApi.ListThirdPartyIntegrations(s.ctx, projectID).Execute()
	return resp, err
}

// AllIntegrations encapsulates the logic to retrieve all integrations from a project all at once.
func (s *Store) AllIntegrations(projectID string) ([]atlasv2.ThirdPartyIntegration, error) {
	return AllPages(func(pageNum, itemsPerPage int) ([]atlasv2.ThirdPartyIntegration, error) {
		page, _, err := s.clientv2.ThirdPartyIntegrationsApi.ListThirdPartyIntegrations(s.ctx, projectID).
			ItemsPerPage(itemsPerPage).
			PageNum(pageNum).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list integrations: %w", err)
		}
		return page.GetResults(), nil
	})
}

type getPageFn[T any] func(int, int) ([]T, error)

func AllPages[T any](getPageFn getPageFn[T]) ([]T, error) {
	allPages := []T{}
	pageNum := 1
	itemsPerPage := MaxAPIPageSize
	for {
		page, err := getPageFn(pageNum, itemsPerPage)
		if err != nil {
			return nil, fmt.Errorf("failed to get all pages: %w", err)
		}
		allPages = append(allPages, page...)
		if len(page) < itemsPerPage {
			break
		}
		pageNum += 1
	}
	return allPages, nil
}
