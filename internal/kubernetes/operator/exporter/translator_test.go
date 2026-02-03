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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestNewTranslator(t *testing.T) {
	scheme, err := NewScheme()
	require.NoError(t, err)

	validCRD := newValidCRD("Group")
	invalidCRDNoVersions := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{},
		},
	}
	invalidCRDNoSchema := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: "v1"}},
		},
	}

	tests := []struct {
		name    string
		crd     *apiextensionsv1.CustomResourceDefinition
		wantErr bool
		errMsg  string
	}{
		{"success", validCRD, false, ""},
		{"nil CRD", nil, true, "CRD is nil"},
		{"no versions", invalidCRDNoVersions, true, "failed to extract CRD version"},
		{"no schema", invalidCRDNoSchema, true, "failed to create translator"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			translator, err := NewTranslator(scheme, tc.crd, SDKVersion)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, translator)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, translator)
			}
		})
	}
}

func TestExtractCRDVersion_MultipleVersions(t *testing.T) {
	// This test verifies behavior not covered by TestNewTranslator:
	// when multiple versions exist, the first one is returned
	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{Name: "v1"},
				{Name: "v2"},
			},
		},
	}

	version, err := extractCRDVersion(crd)

	require.NoError(t, err)
	assert.Equal(t, "v1", version)
}

// newValidCRD creates a valid CRD for testing.
func newValidCRD(kind string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "atlas.generated.mongodb.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: kind},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name: "v1",
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"spec": {
								Type: "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{
									SDKVersion: {Type: "object"},
								},
							},
						},
					},
				},
			}},
		},
	}
}
