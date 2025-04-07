// Copyright 2025 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package features_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/crds"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

const (
	urlTemplate    = "https://raw.githubusercontent.com/mongodb/mongodb-atlas-kubernetes/v%s/bundle/manifests/atlas.mongodb.com_%s.yaml"
	requestTimeout = 10 * time.Second
)

func TestEmbeddedCRDs(t *testing.T) {
	client := http.Client{}
	provider := crds.EmbeddedAtlasCRDProvider{}
	diffs := []string{}
	for _, version := range features.SupportedVersions() {
		resources, ok := features.GetResourcesForVersion(version)
		require.True(t, ok)
		for _, resource := range resources {
			if diffResource(t, &client, provider, version, resource) {
				diffs = append(diffs, fmt.Sprintf("%s/%s", version, resource))
			}
		}
	}
	assert.Equal(t, []string{}, diffs)
}

func diffResource(t *testing.T, httpClient *http.Client, provider crds.AtlasOperatorCRDProvider, version, resource string) bool {
	localCRD, err := provider.GetAtlasOperatorResource(resource, version)
	require.NoError(t, err)
	remoteCRD, err := getGitHubCRD(httpClient, resource, version)
	require.NoError(t, err)
	return !reflect.DeepEqual(localCRD, remoteCRD)
}

func getGitHubCRD(httpClient *http.Client, resource, version string) (*apiextensionsv1.CustomResourceDefinition, error) {
	ctx, cancelF := context.WithTimeout(context.Background(), requestTimeout)
	defer cancelF()

	url := fmt.Sprintf(urlTemplate, version, resource)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request GET %s: %w", url, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to GET %s: %w", url, err)
	}
	//nolint:errcheck
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from GET %s: %w", url, err)
	}

	decoded := &apiextensionsv1.CustomResourceDefinition{}
	err = yaml.Unmarshal(data, decoded)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
