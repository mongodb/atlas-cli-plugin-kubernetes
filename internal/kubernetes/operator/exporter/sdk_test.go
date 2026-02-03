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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSDKClient_Success(t *testing.T) {
	tests := []struct {
		name    string
		service string
	}{
		{
			name:    "cloud service",
			service: "cloud",
		},
		{
			name:    "cloudgov service",
			service: config.CloudGovService,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewSDKClient(tc.service)

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

	client, err := NewSDKClient("cloud")

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "http client error")
}
