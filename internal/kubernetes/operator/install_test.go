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

//go:build unit

package operator

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/mocks"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/atlas-sdk/v20250312006/admin"
)

func TestInstall_addAPIKeyIPAccessList(t *testing.T) {
	tests := map[string]struct {
		ipAccessList string
		expectedErr  error
	}{
		"An IP is provided": {
			ipAccessList: "104.30.164.5",
			expectedErr:  nil,
		},
		"An CIDR is provided": {
			ipAccessList: "192.168.100.177/24",
			expectedErr:  nil,
		},
		"Multiple entries are provided": {
			ipAccessList: "104.30.164.5,192.168.100.177/24",
			expectedErr:  nil,
		},
		"API failed to add ip access list": {
			ipAccessList: "104.30.164.5,192.168.100.177/24",
			expectedErr:  fmt.Errorf("failed to add IP access list to API key: %w", errors.New("failed to add IP access list")),
		},
	}
	for name, tt := range tests {
		storeMock := mocks.NewMockOperatorGenericStore(gomock.NewController(t))
		storeMock.EXPECT().AddIPAccessList("orgID", "apiKeyID", gomock.Any()).
			DoAndReturn(func(string, string, *[]admin.UserAccessListRequest) error {
				if tt.expectedErr != nil {
					return errors.New("failed to add IP access list")
				}

				return nil
			}).
			Times(1)

		t.Run(name, func(t *testing.T) {
			i := &Install{
				ipAccessList: tt.ipAccessList,
				atlasStore:   storeMock,
			}
			err := i.addAPIKeyIPAccessList("orgID", "apiKeyID")
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
