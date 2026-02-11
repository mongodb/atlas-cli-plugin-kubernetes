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

package operator

import (
	"context"
	"errors"
	"testing"

	"github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/pointer"
	samplesv1 "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/crapi/testdata/samples/v1"
	atlasauth "go.mongodb.org/atlas/auth"

	generated "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/exporter/generated"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// mockExporter is a simple mock implementation of generated.Exporter for testing.
type mockExporter struct {
	objects            []client.Object
	err                error
	receivedRefObjects []client.Object
}

func (m *mockExporter) Export(_ context.Context, referencedObjects []client.Object) ([]client.Object, error) {
	m.receivedRefObjects = referencedObjects
	return m.objects, m.err
}

// mockCredentialsProvider is a mock implementation of store.CredentialsGetter for testing.
type mockCredentialsProvider struct {
	publicKey  string
	privateKey string
}

func (m *mockCredentialsProvider) PublicAPIKey() string             { return m.publicKey }
func (m *mockCredentialsProvider) PrivateAPIKey() string            { return m.privateKey }
func (m *mockCredentialsProvider) ClientID() string                 { return "" }
func (m *mockCredentialsProvider) ClientSecret() string             { return "" }
func (m *mockCredentialsProvider) Token() (*atlasauth.Token, error) { return nil, nil }
func (m *mockCredentialsProvider) AuthType() config.AuthMechanism   { return config.APIKeys }

var _ store.CredentialsGetter = (*mockCredentialsProvider)(nil)

// newTestScheme creates a scheme with the required types for testing.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, samplesv1.AddToScheme(scheme))
	return scheme
}

// newGroup creates a Group CRD object for testing.
func newGroup(projectName, orgID string) *samplesv1.Group {
	return &samplesv1.Group{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "atlas.generated.mongodb.com/v1",
			Kind:       "Group",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "group-placeholder",
		},
		Spec: samplesv1.GroupSpec{
			V20250312: &samplesv1.GroupSpecV20250312{
				Entry: &samplesv1.GroupSpecV20250312Entry{
					Name:  projectName,
					OrgId: orgID,
				},
			},
		},
	}
}

// newFlexCluster creates a FlexCluster CRD object for testing.
func newFlexCluster(clusterName, groupID, provider, region string) *samplesv1.FlexCluster {
	return &samplesv1.FlexCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "atlas.generated.mongodb.com/v1",
			Kind:       "FlexCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "flexcluster-placeholder",
		},
		Spec: samplesv1.FlexClusterSpec{
			V20250312: &samplesv1.FlexClusterSpecV20250312{
				GroupId: pointer.Get(groupID),
				Entry: &samplesv1.FlexClusterSpecV20250312Entry{
					Name: clusterName,
					ProviderSettings: samplesv1.ProviderSettings{
						BackingProviderName: provider,
						RegionName:          region,
					},
				},
			},
		},
	}
}

func TestGeneratedExporterRun(t *testing.T) {
	tests := []struct {
		name                 string
		targetNamespace      string
		exporters            func() ([]generated.Exporter, []*mockExporter)
		independentResources bool
		includeSecrets       bool
		credentialsProvider  store.CredentialsGetter
		orgID                string
		wantErr              bool
		errContains          string
		validate             func(t *testing.T, output string, exporters []*mockExporter)
	}{
		{
			name:            "exports Group and FlexCluster with namespace",
			targetNamespace: "atlas-operator",
			exporters: func() ([]generated.Exporter, []*mockExporter) {
				groupExp := &mockExporter{objects: []client.Object{newGroup("my-atlas-project", "507f1f77bcf86cd799439011")}}
				flexExp := &mockExporter{objects: []client.Object{newFlexCluster("my-flex-cluster", "507f1f77bcf86cd799439011", "AWS", "US_EAST_1")}}
				return []generated.Exporter{groupExp, flexExp}, []*mockExporter{groupExp, flexExp}
			},
			validate: func(t *testing.T, output string, _ []*mockExporter) {
				// Verify Group is exported
				assert.Contains(t, output, "kind: Group")
				assert.Contains(t, output, "name: group-my-atlas-project")
				assert.Contains(t, output, "namespace: atlas-operator")

				// Verify FlexCluster is exported
				assert.Contains(t, output, "kind: FlexCluster")
				assert.Contains(t, output, "name: flexcluster-my-flex-cluster")

				// Verify status field is empty
				assert.Contains(t, output, "status: {}")

				// Verify no secrets when not requested
				assert.NotContains(t, output, "kind: Secret")
				assert.NotContains(t, output, "connectionSecretRef:")

			},
		},
		{
			name:            "returns error from exporter",
			targetNamespace: "default",
			exporters: func() ([]generated.Exporter, []*mockExporter) {
				exp := &mockExporter{err: errors.New("atlas API unavailable")}
				return []generated.Exporter{exp}, []*mockExporter{exp}
			},
			wantErr:     true,
			errContains: "failed to export resources",
		},
		{
			name:                 "independentResources exports Group and FlexCluster without cross-references",
			targetNamespace:      "atlas-operator",
			independentResources: true,
			exporters: func() ([]generated.Exporter, []*mockExporter) {
				groupExp := &mockExporter{objects: []client.Object{newGroup("production-project", "507f1f77bcf86cd799439011")}}
				flexExp := &mockExporter{objects: []client.Object{newFlexCluster("prod-cluster", "507f1f77bcf86cd799439011", "GCP", "US_CENTRAL1")}}
				return []generated.Exporter{groupExp, flexExp}, []*mockExporter{groupExp, flexExp}
			},
			validate: func(t *testing.T, output string, exporters []*mockExporter) {
				// Verify both resources exported
				assert.Contains(t, output, "kind: Group")
				assert.Contains(t, output, "kind: FlexCluster")

				// Both exporters should receive nil/empty referenced objects (independent)
				assert.Empty(t, exporters[0].receivedRefObjects)
				assert.Empty(t, exporters[1].receivedRefObjects)

				// When independentResources is true, secrets should be included
				assert.Contains(t, output, "kind: Secret")
				assert.Contains(t, output, "connectionSecretRef:")
			},
		},
		{
			name:                 "independentResources is false when exporting",
			targetNamespace:      "atlas-operator",
			independentResources: false,
			exporters: func() ([]generated.Exporter, []*mockExporter) {
				groupExp := &mockExporter{objects: []client.Object{newGroup("shared-project", "507f1f77bcf86cd799439011")}}
				flexExp := &mockExporter{objects: []client.Object{newFlexCluster("shared-cluster", "507f1f77bcf86cd799439011", "AZURE", "US_EAST_2")}}
				return []generated.Exporter{groupExp, flexExp}, []*mockExporter{groupExp, flexExp}
			},
			validate: func(t *testing.T, output string, exporters []*mockExporter) {
				assert.Contains(t, output, "kind: Group")
				assert.Contains(t, output, "kind: FlexCluster")

				// First exporter (Group) receives empty - no previous objects
				assert.Empty(t, exporters[0].receivedRefObjects)

				// Second exporter (FlexCluster) receives Group for reference resolution
				require.Len(t, exporters[1].receivedRefObjects, 1)
				assert.Equal(t, "group-shared-project", exporters[1].receivedRefObjects[0].GetName())

				// No secrets when both flags false
				assert.NotContains(t, output, "kind: Secret")
			},
		},
		{
			name:            "includeSecrets generates credentials secret for resources",
			targetNamespace: "atlas-operator",
			includeSecrets:  true,
			credentialsProvider: &mockCredentialsProvider{
				publicKey:  "ABCDEFGH",
				privateKey: "12345678-1234-1234-1234-123456789012",
			},
			orgID: "507f1f77bcf86cd799439011",
			exporters: func() ([]generated.Exporter, []*mockExporter) {
				groupExp := &mockExporter{objects: []client.Object{newGroup("secure-project", "507f1f77bcf86cd799439011")}}
				flexExp := &mockExporter{objects: []client.Object{newFlexCluster("secure-cluster", "507f1f77bcf86cd799439011", "AWS", "EU_WEST_1")}}
				return []generated.Exporter{groupExp, flexExp}, []*mockExporter{groupExp, flexExp}
			},
			validate: func(t *testing.T, output string, _ []*mockExporter) {
				// Verify secret is generated
				assert.Contains(t, output, "kind: Secret")
				assert.Contains(t, output, "name: atlas-credentials")

				// Verify resources reference the secret
				assert.Contains(t, output, "connectionSecretRef:")

				assert.Contains(t, output, "kind: Group")
				assert.Contains(t, output, "kind: FlexCluster")
			},
		},
		{
			name:                 "independentResources without includeSecrets still generates placeholder secret",
			targetNamespace:      "atlas-operator",
			includeSecrets:       false,
			independentResources: true,
			orgID:                "507f1f77bcf86cd799439011",
			exporters: func() ([]generated.Exporter, []*mockExporter) {
				groupExp := &mockExporter{objects: []client.Object{newGroup("standalone-project", "507f1f77bcf86cd799439011")}}
				return []generated.Exporter{groupExp}, []*mockExporter{groupExp}
			},
			validate: func(t *testing.T, output string, _ []*mockExporter) {
				// Secret should be generated (for standalone operation)
				assert.Contains(t, output, "kind: Secret")
				assert.Contains(t, output, "name: atlas-credentials")
				assert.Contains(t, output, "connectionSecretRef:")
			},
		},
		{
			name:                 "no secret when both includeSecrets and independentResources are false",
			targetNamespace:      "atlas-operator",
			includeSecrets:       false,
			independentResources: false,
			orgID:                "507f1f77bcf86cd799439011",
			exporters: func() ([]generated.Exporter, []*mockExporter) {
				groupExp := &mockExporter{objects: []client.Object{newGroup("basic-project", "507f1f77bcf86cd799439011")}}
				flexExp := &mockExporter{objects: []client.Object{newFlexCluster("basic-cluster", "507f1f77bcf86cd799439011", "AWS", "US_WEST_2")}}
				return []generated.Exporter{groupExp, flexExp}, []*mockExporter{groupExp, flexExp}
			},
			validate: func(t *testing.T, output string, _ []*mockExporter) {
				assert.NotContains(t, output, "kind: Secret")
				assert.Contains(t, output, "kind: Group")
				assert.Contains(t, output, "kind: FlexCluster")
				assert.NotContains(t, output, "connectionSecretRef:")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scheme := newTestScheme(t)
			exporters, mockExps := tc.exporters()

			exp := NewGeneratedExporter(GeneratedExporterConfig{
				TargetNamespace:      tc.targetNamespace,
				Scheme:               scheme,
				Exporters:            exporters,
				IndependentResources: tc.independentResources,
				IncludeSecrets:       tc.includeSecrets,
				CredentialsProvider:  tc.credentialsProvider,
				OrgID:                tc.orgID,
			})

			output, err := exp.Run()

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				assert.Empty(t, output)
			} else {
				require.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, output, mockExps)
				}
			}
		})
	}
}
