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

	"go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_alert_configuration.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store AlertConfigurationLister

type AlertConfigurationLister interface {
	AlertConfigurations(projectID string) ([]admin.GroupAlertsConfig, error)
}

// AlertConfigurations encapsulate the logic to manage different cloud providers.
func (s *Store) AlertConfigurations(projectID string) ([]admin.GroupAlertsConfig, error) {
	return AllPages(func(pageNum, itemsPerPage int) ([]admin.GroupAlertsConfig, error) {
		page, _, err := s.clientv2.AlertConfigurationsApi.ListAlertConfigurations(s.ctx, projectID).
			PageNum(pageNum).
			ItemsPerPage(itemsPerPage).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list project alert configurations: %w", err)
		}

		return page.GetResults(), nil
	})
}
