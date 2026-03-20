// Copyright 2025 MongoDB Inc
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

package exporter

import (
	"errors"
	"net/http"
	"testing"

	"github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockServiceGetter implements store.ServiceGetter for testing.
type mockServiceGetter struct {
	service       string
	opsManagerURL string
}

func (m *mockServiceGetter) Service() string {
	return m.service
}

func (m *mockServiceGetter) OpsManagerURL() string {
	return m.opsManagerURL
}

// Verify mockServiceGetter implements store.ServiceGetter
var _ store.ServiceGetter = (*mockServiceGetter)(nil)

func TestNewSDKClient_Success(t *testing.T) {
	tests := []struct {
		name    string
		profile store.ServiceGetter
	}{
		{
			name:    "cloud service",
			profile: &mockServiceGetter{service: "cloud"},
		},
		{
			name:    "cloudgov service",
			profile: &mockServiceGetter{service: config.CloudGovService},
		},
		{
			name: "ops manager URL takes precedence",
			profile: &mockServiceGetter{
				service:       config.CloudGovService,
				opsManagerURL: "https://ops.example.com/",
			},
		},
		{
			name: "ops manager URL only",
			profile: &mockServiceGetter{
				opsManagerURL: "https://ops.example.com/",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewSDKClient(tc.profile)

			require.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}

func TestNewSDKClient_HTTPClientError(t *testing.T) {
	// Save original and restore after test
	originalFunc := httpClientFunc
	defer func() { httpClientFunc = originalFunc }()

	httpClientFunc = func() (*http.Client, error) {
		return nil, errors.New("http client error")
	}

	client, err := NewSDKClient(&mockServiceGetter{service: "cloud"})

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "http client error")
}
