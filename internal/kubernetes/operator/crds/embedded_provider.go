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

package crds

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"path"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	k8syaml "sigs.k8s.io/yaml"
)

//go:embed generated/*/crds.yaml
var generatedCRDsFS embed.FS

// EmbeddedAtlasCRDProvider provides CRDs from embedded files organized by operator version.
// This is used for generated CRDs that are not yet publicly available on GitHub.
type EmbeddedAtlasCRDProvider struct {
	// crds maps version -> resource name -> CRD definition
	crds map[string]map[string]*apiextensionsv1.CustomResourceDefinition
}

// NewEmbeddedAtlasCRDProvider creates a new provider that loads CRDs from embedded files.
func NewEmbeddedAtlasCRDProvider() (*EmbeddedAtlasCRDProvider, error) {
	provider := &EmbeddedAtlasCRDProvider{
		crds: make(map[string]map[string]*apiextensionsv1.CustomResourceDefinition),
	}

	if err := provider.loadAllVersions(); err != nil {
		return nil, fmt.Errorf("failed to load embedded CRDs: %w", err)
	}

	return provider, nil
}

// loadAllVersions discovers and loads all versioned CRD files.
func (p *EmbeddedAtlasCRDProvider) loadAllVersions() error {
	entries, err := generatedCRDsFS.ReadDir("generated")
	if err != nil {
		return fmt.Errorf("failed to read generated directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		if err := p.loadCRDsForVersion(version); err != nil {
			return fmt.Errorf("failed to load CRDs for version %s: %w", version, err)
		}
	}

	return nil
}

// loadCRDsForVersion reads and parses the CRDs file for a specific version.
func (p *EmbeddedAtlasCRDProvider) loadCRDsForVersion(version string) error {
	filePath := path.Join("generated", version, "crds.yaml")
	data, err := generatedCRDsFS.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read CRDs file for version %s: %w", version, err)
	}

	p.crds[version] = make(map[string]*apiextensionsv1.CustomResourceDefinition)

	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))

	for {
		doc, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read YAML document: %w", err)
		}

		// Skip empty documents and comments
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := k8syaml.Unmarshal(doc, crd); err != nil {
			// Skip non-CRD documents (e.g., comments)
			continue
		}

		// Only store valid CRDs
		if crd.Name != "" {
			p.crds[version][crd.Name] = crd
		}
	}

	return nil
}

// GetAtlasOperatorResource returns the CRD for the given resource name and operator version.
func (p *EmbeddedAtlasCRDProvider) GetAtlasOperatorResource(resourceName, version string) (*apiextensionsv1.CustomResourceDefinition, error) {
	versionCRDs, ok := p.crds[version]
	if !ok {
		return nil, fmt.Errorf("operator version %q not found in embedded resources", version)
	}

	crd, ok := versionCRDs[resourceName]
	if !ok {
		return nil, fmt.Errorf("CRD %q not found for operator version %q", resourceName, version)
	}

	return crd, nil
}
