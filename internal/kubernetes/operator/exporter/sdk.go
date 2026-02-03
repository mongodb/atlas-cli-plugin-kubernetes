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

package exporter

import (
	"net/http"

	"github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-core/transport"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/log"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/version"
	admin "go.mongodb.org/atlas-sdk/v20250312013/admin"
)

const (
	cloudGovServiceURL = "https://cloud.mongodbgov.com/"

	// SDKVersion is the Atlas SDK major version used by this exporter.
	// This must match the imported SDK package version (v20250312013).
	SDKVersion = "v20250312"
)

// httpClientFunc is a function variable for dependency injection in tests.
var httpClientFunc = func() (*http.Client, error) {
	return transport.HTTPClient(version.Version, transport.Default())
}

// NewSDKClient creates a new Atlas SDK v20250312013 client using the same
// authentication and configuration pattern as the Store abstraction.
func NewSDKClient(service string) (*admin.APIClient, error) {
	httpClient, err := httpClientFunc()
	if err != nil {
		return nil, err
	}

	opts := []admin.ClientModifier{
		admin.UseHTTPClient(httpClient),
		admin.UseUserAgent(config.UserAgent(version.Version)),
		admin.UseDebug(log.IsDebugLevel()),
	}

	if service == config.CloudGovService {
		opts = append(opts, admin.UseBaseURL(cloudGovServiceURL))
	}

	return admin.NewClient(opts...)
}
