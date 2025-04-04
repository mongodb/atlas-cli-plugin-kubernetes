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
	"embed"
	"fmt"
	"io"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

const (
	embeddedBase = "versions/v%s/atlas.mongodb.com_%s.yaml"
)

//go:generate mockgen -destination=../../../mocks/mock_atlas_operator_crd_provider.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/crds AtlasOperatorCRDProvider
type AtlasOperatorCRDProvider interface {
	GetAtlasOperatorResource(resourceName, version string) (*apiextensionsv1.CustomResourceDefinition, error)
}

//go:embed versions/*
var crdVersions embed.FS

type EmbeddedAtlasCRDProvider struct{}

func (p EmbeddedAtlasCRDProvider) GetAtlasOperatorResource(resourceName, version string) (*apiextensionsv1.CustomResourceDefinition, error) {
	embeddedPath := fmt.Sprintf(embeddedBase, version, resourceName)
	f, err := crdVersions.Open(embeddedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open embeddedPath path %q: %w", embeddedPath, err)
	}
	//nolint:errcheck
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read embeddedPath file %q: %w", embeddedPath, err)
	}

	decoded := &apiextensionsv1.CustomResourceDefinition{}
	err = yaml.Unmarshal(data, decoded)
	if err != nil {
		return nil, fmt.Errorf("failed to umarshall embeddedPath file %q: %w", embeddedPath, err)
	}

	return decoded, nil
}
