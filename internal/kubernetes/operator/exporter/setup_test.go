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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/mocks"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admin "go.mongodb.org/atlas-sdk/v20250312013/admin"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// testProfile implements store.ServiceGetter for testing.
type testProfile struct {
	service       string
	opsManagerURL string
}

func (p *testProfile) Service() string {
	return p.service
}

func (p *testProfile) OpsManagerURL() string {
	return p.opsManagerURL
}

// Verify testProfile implements store.ServiceGetter
var _ store.ServiceGetter = (*testProfile)(nil)

func TestSetup_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockAtlasOperatorCRDProvider(ctrl)

	// Expect calls for all supported resources
	for _, resource := range SupportedResources {
		mockProvider.EXPECT().
			GetAtlasOperatorResource(resource.CRDName, "2.0.0").
			Return(newTestCRD(resource.CRDName), nil)
	}

	cfg := SetupConfig{
		ProjectID:       "test-project-id",
		TargetNamespace: "test-namespace",
		Profile:         &testProfile{service: "cloud"},
		CRDProvider:     mockProvider,
		OperatorVersion: "2.0.0",
	}

	exporter, err := Setup(cfg)

	require.NoError(t, err)
	assert.NotNil(t, exporter)
}

func TestSetup_SDKClientError(t *testing.T) {
	// Save original and restore after test
	originalFunc := newSDKClientFunc
	defer func() { newSDKClientFunc = originalFunc }()

	newSDKClientFunc = func(_ store.ServiceGetter) (*admin.APIClient, error) {
		return nil, errors.New("SDK client error")
	}

	cfg := SetupConfig{
		ProjectID:       "test-project-id",
		TargetNamespace: "test-namespace",
		Profile:         &testProfile{service: "cloud"},
		CRDProvider:     nil,
		OperatorVersion: "2.0.0",
	}

	exporter, err := Setup(cfg)

	require.Error(t, err)
	assert.Nil(t, exporter)
	assert.Contains(t, err.Error(), "failed to create SDK client")
}

func TestSetup_SchemeError(t *testing.T) {
	// Save originals and restore after test
	originalSDKFunc := newSDKClientFunc
	originalSchemeFunc := newSchemeFunc
	defer func() {
		newSDKClientFunc = originalSDKFunc
		newSchemeFunc = originalSchemeFunc
	}()

	newSDKClientFunc = func(_ store.ServiceGetter) (*admin.APIClient, error) {
		return &admin.APIClient{}, nil
	}
	newSchemeFunc = func() (*runtime.Scheme, error) {
		return nil, errors.New("scheme error")
	}

	cfg := SetupConfig{
		ProjectID:       "test-project-id",
		TargetNamespace: "test-namespace",
		Profile:         &testProfile{service: "cloud"},
		CRDProvider:     nil,
		OperatorVersion: "2.0.0",
	}

	exporter, err := Setup(cfg)

	require.Error(t, err)
	assert.Nil(t, exporter)
	assert.Contains(t, err.Error(), "failed to create scheme")
}

func TestSetup_CRDErrors(t *testing.T) {
	invalidCRD := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{},
		},
	}

	tests := []struct {
		name       string
		crd        *apiextensionsv1.CustomResourceDefinition
		crdErr     error
		wantErrMsg string
	}{
		{
			name:       "CRD fetch fails",
			crd:        nil,
			crdErr:     errors.New("fetch error"),
			wantErrMsg: "failed to fetch CRD",
		},
		{
			name:       "translator creation fails with invalid CRD",
			crd:        invalidCRD,
			crdErr:     nil,
			wantErrMsg: "failed to create translator",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := mocks.NewMockAtlasOperatorCRDProvider(ctrl)
			mockProvider.EXPECT().
				GetAtlasOperatorResource(SupportedResources[0].CRDName, "2.0.0").
				Return(tc.crd, tc.crdErr)

			cfg := SetupConfig{
				ProjectID:       "test-project-id",
				TargetNamespace: "test-namespace",
				Profile:         &testProfile{service: "cloud"},
				CRDProvider:     mockProvider,
				OperatorVersion: "2.0.0",
			}

			// newSDKClientFunc and newSchemeFunc must succeed for CRD/translator errors to be reached
			originalSDKFunc := newSDKClientFunc
			originalSchemeFunc := newSchemeFunc
			defer func() {
				newSDKClientFunc = originalSDKFunc
				newSchemeFunc = originalSchemeFunc
			}()
			newSDKClientFunc = func(_ store.ServiceGetter) (*admin.APIClient, error) { return &admin.APIClient{}, nil }
			newSchemeFunc = func() (*runtime.Scheme, error) { return runtime.NewScheme(), nil }

			exporter, err := Setup(cfg)

			require.Error(t, err)
			assert.Nil(t, exporter)
			assert.Contains(t, err.Error(), tc.wantErrMsg)
		})
	}
}

// newTestCRD creates a valid CRD for testing with the required structure.
func newTestCRD(name string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "atlas.generated.mongodb.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind: extractKindFromCRDName(name),
			},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name: "v1",
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"spec": {
									Type: "object",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										SDKVersion: {
											Type: "object",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// extractKindFromCRDName extracts the Kind from a CRD name.
// e.g., "groups.atlas.generated.mongodb.com" -> "Group"
func extractKindFromCRDName(name string) string {
	kindMap := map[string]string{
		"groups.atlas.generated.mongodb.com":        "Group",
		"clusters.atlas.generated.mongodb.com":      "Cluster",
		"flexclusters.atlas.generated.mongodb.com":  "FlexCluster",
		"databaseusers.atlas.generated.mongodb.com": "DatabaseUser",
	}
	if kind, ok := kindMap[name]; ok {
		return kind
	}
	return "Unknown"
}
