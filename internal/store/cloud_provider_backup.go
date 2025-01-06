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
	atlasClustersPinned "go.mongodb.org/atlas-sdk/v20240530005/admin"
)

//go:generate mockgen -destination=../mocks/mock_cloud_provider_backup.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store ScheduleDescriber

type ScheduleDescriber interface {
	DescribeSchedule(string, string) (*atlasClustersPinned.DiskBackupSnapshotSchedule, error)
}

// DescribeSchedule encapsulates the logic to manage different cloud providers.
func (s *Store) DescribeSchedule(projectID, clusterName string) (*atlasClustersPinned.DiskBackupSnapshotSchedule, error) {
	result, _, err := s.clientClusters.CloudBackupsApi.GetBackupSchedule(s.ctx, projectID, clusterName).Execute()
	return result, err
}
