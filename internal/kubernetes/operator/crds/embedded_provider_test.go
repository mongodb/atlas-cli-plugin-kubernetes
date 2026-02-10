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

//go:build unit

package crds

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOperatorVersion is the operator version used in tests.
// This should match an embedded version directory.
const testOperatorVersion = "2.13.0"

func TestNewEmbeddedAtlasCRDProvider(t *testing.T) {
	provider, err := NewEmbeddedAtlasCRDProvider()

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.NotEmpty(t, provider.crds, "should load CRDs from embedded files")
}

func TestNewEmbeddedAtlasCRDProvider_ReadDirError(t *testing.T) {
	// Save and restore original FS
	originalFS := crdFS
	defer func() { crdFS = originalFS }()

	// Use an empty FS that doesn't have "generated" directory
	crdFS = fstest.MapFS{}

	provider, err := NewEmbeddedAtlasCRDProvider()

	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "failed to load embedded CRDs")
}

func TestNewEmbeddedAtlasCRDProvider_ReadFileError(t *testing.T) {
	// Save and restore original FS
	originalFS := crdFS
	defer func() { crdFS = originalFS }()

	// Create a FS with a version directory but no crds.yaml file
	crdFS = fstest.MapFS{
		"generated/1.0.0": &fstest.MapFile{Mode: fs.ModeDir},
	}

	provider, err := NewEmbeddedAtlasCRDProvider()

	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "failed to load CRDs for version")
}

func TestNewEmbeddedAtlasCRDProvider_SkipsNonDirectory(t *testing.T) {
	// Save and restore original FS
	originalFS := crdFS
	defer func() { crdFS = originalFS }()

	// Create a FS with a file in generated directory (should be skipped)
	validCRD := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test.atlas.generated.mongodb.com
spec:
  group: atlas.generated.mongodb.com
  names:
    kind: Test
    listKind: TestList
    plural: tests
    singular: test
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
`

	crdFS = fstest.MapFS{
		"generated/readme.txt":      &fstest.MapFile{Data: []byte("not a directory")},
		"generated/1.0.0":           &fstest.MapFile{Mode: fs.ModeDir},
		"generated/1.0.0/crds.yaml": &fstest.MapFile{Data: []byte(validCRD)},
	}

	provider, err := NewEmbeddedAtlasCRDProvider()

	require.NoError(t, err)
	require.NotNil(t, provider)
	// Should have loaded the valid version
	_, ok := provider.crds["1.0.0"]
	assert.True(t, ok, "should have loaded version 1.0.0")
}

func TestLoadCRDsForVersion_EmptyDocument(t *testing.T) {
	// Save and restore original FS
	originalFS := crdFS
	defer func() { crdFS = originalFS }()

	// Create a FS with a CRD file that has empty documents
	yamlWithEmptyDocs := `---
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test.atlas.generated.mongodb.com
spec:
  group: atlas.generated.mongodb.com
  names:
    kind: Test
    listKind: TestList
    plural: tests
    singular: test
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
---
`

	crdFS = fstest.MapFS{
		"generated/1.0.0":           &fstest.MapFile{Mode: fs.ModeDir},
		"generated/1.0.0/crds.yaml": &fstest.MapFile{Data: []byte(yamlWithEmptyDocs)},
	}

	provider, err := NewEmbeddedAtlasCRDProvider()

	require.NoError(t, err)
	require.NotNil(t, provider)
	// Should have loaded the valid CRD despite empty documents
	crd, ok := provider.crds["1.0.0"]["test.atlas.generated.mongodb.com"]
	assert.True(t, ok, "should have loaded the CRD")
	assert.Equal(t, "Test", crd.Spec.Names.Kind)
}

func TestLoadCRDsForVersion_InvalidYAML(t *testing.T) {
	// Save and restore original FS
	originalFS := crdFS
	defer func() { crdFS = originalFS }()

	// Create a FS with invalid YAML (but not a parse error, just not a valid CRD)
	invalidCRD := `---
notACRD: true
someField: value
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: valid.atlas.generated.mongodb.com
spec:
  group: atlas.generated.mongodb.com
  names:
    kind: Valid
    listKind: ValidList
    plural: valids
    singular: valid
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
`

	crdFS = fstest.MapFS{
		"generated/1.0.0":           &fstest.MapFile{Mode: fs.ModeDir},
		"generated/1.0.0/crds.yaml": &fstest.MapFile{Data: []byte(invalidCRD)},
	}

	provider, err := NewEmbeddedAtlasCRDProvider()

	require.NoError(t, err)
	require.NotNil(t, provider)
	// Should have loaded only the valid CRD (skipping the invalid one)
	_, ok := provider.crds["1.0.0"]["valid.atlas.generated.mongodb.com"]
	assert.True(t, ok, "should have loaded the valid CRD")
}

func TestEmbeddedAtlasCRDProvider_GetAtlasOperatorResource(t *testing.T) {
	provider, err := NewEmbeddedAtlasCRDProvider()
	require.NoError(t, err)

	version := testOperatorVersion

	tests := []struct {
		name         string
		resourceName string
		version      string
		wantErr      bool
		wantKind     string
		errContains  string
	}{
		{
			name:         "groups CRD exists",
			resourceName: "groups.atlas.generated.mongodb.com",
			version:      version,
			wantErr:      false,
			wantKind:     "Group",
		},
		{
			name:         "clusters CRD exists",
			resourceName: "clusters.atlas.generated.mongodb.com",
			version:      version,
			wantErr:      false,
			wantKind:     "Cluster",
		},
		{
			name:         "flexclusters CRD exists",
			resourceName: "flexclusters.atlas.generated.mongodb.com",
			version:      version,
			wantErr:      false,
			wantKind:     "FlexCluster",
		},
		{
			name:         "databaseusers CRD exists",
			resourceName: "databaseusers.atlas.generated.mongodb.com",
			version:      version,
			wantErr:      false,
			wantKind:     "DatabaseUser",
		},
		{
			name:         "organizations CRD exists",
			resourceName: "organizations.atlas.generated.mongodb.com",
			version:      version,
			wantErr:      false,
			wantKind:     "Organization",
		},
		{
			name:         "non-existent CRD returns error",
			resourceName: "nonexistent.atlas.generated.mongodb.com",
			version:      version,
			wantErr:      true,
			errContains:  "not found",
		},
		{
			name:         "non-existent version returns error",
			resourceName: "groups.atlas.generated.mongodb.com",
			version:      "0.0.0",
			wantErr:      true,
			errContains:  "not found in embedded resources",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			crd, err := provider.GetAtlasOperatorResource(tc.resourceName, tc.version)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, crd)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, crd)
				assert.Equal(t, tc.resourceName, crd.Name)
				assert.Equal(t, tc.wantKind, crd.Spec.Names.Kind)
				assert.Equal(t, "atlas.generated.mongodb.com", crd.Spec.Group)
			}
		})
	}
}
