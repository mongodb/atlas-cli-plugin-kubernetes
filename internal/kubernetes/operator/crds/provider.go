// Copyright 2023 MongoDB Inc
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

package crds

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

const (
	urlTemplate    = "https://raw.githubusercontent.com/mongodb/mongodb-atlas-kubernetes/v%s/bundle/manifests/atlas.mongodb.com_%s.yaml"
	requestTimeout = 10 * time.Second
)

//go:generate mockgen -destination=../../../mocks/mock_atlas_operator_crd_provider.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/crds AtlasOperatorCRDProvider
type AtlasOperatorCRDProvider interface {
	GetAtlasOperatorResource(resourceName, version string) (*apiextensionsv1.CustomResourceDefinition, error)
}

type GithubAtlasCRDProvider struct {
	client *http.Client
}

func NewGithubAtlasCRDProvider() *GithubAtlasCRDProvider {
	return &GithubAtlasCRDProvider{client: &http.Client{}}
}

func (p *GithubAtlasCRDProvider) GetAtlasOperatorResource(resourceName, version string) (res *apiextensionsv1.CustomResourceDefinition, err error) {
	ctx, cancelF := context.WithTimeout(context.Background(), requestTimeout)
	defer cancelF()

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(urlTemplate, version, resourceName), nil)
	if reqErr != nil {
		return nil, reqErr
	}

	resp, respErr := p.client.Do(req)
	if respErr != nil {
		return nil, respErr
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
	}(resp.Body)

	data, dtErr := io.ReadAll(resp.Body)
	if dtErr != nil {
		return nil, dtErr
	}

	decoded := &apiextensionsv1.CustomResourceDefinition{}
	err = yaml.Unmarshal(data, decoded)

	return decoded, err
}
