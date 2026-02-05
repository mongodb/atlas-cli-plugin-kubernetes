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
	"testing"

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
