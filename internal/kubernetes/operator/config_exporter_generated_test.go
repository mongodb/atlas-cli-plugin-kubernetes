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
	"reflect"
	"strings"
	"testing"

	"github.com/crd2go/crd2go/k8s"
	"github.com/mongodb/atlas-cli-core/config"
	atlasauth "go.mongodb.org/atlas/auth"

	generated "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/exporter/generated"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func TestNewGeneratedExporter(t *testing.T) {
	scheme := runtime.NewScheme()
	exporters := []generated.Exporter{&mockExporter{}}

	exp := NewGeneratedExporter(GeneratedExporterConfig{
		TargetNamespace:      "test-namespace",
		Scheme:               scheme,
		Exporters:            exporters,
		IndependentResources: true,
		IncludeSecrets:       true,
		OrgID:                "test-org",
	})

	require.NotNil(t, exp)
	assert.Equal(t, "test-namespace", exp.targetNamespace)
	assert.Equal(t, scheme, exp.scheme)
	assert.Len(t, exp.exporters, 1)
	assert.True(t, exp.independentResources)
	assert.True(t, exp.includeSecrets)
}

func TestGeneratedExporter_Run(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	// Helper to create a fresh ConfigMap for each test
	newConfigMap := func() *corev1.ConfigMap {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-config",
			},
			Data: map[string]string{"key": "value"},
		}
	}

	tests := []struct {
		name            string
		targetNamespace string
		exporters       []generated.Exporter
		wantErr         bool
		errMsg          string
		validate        func(t *testing.T, output string)
	}{
		{
			name:            "success with objects and namespace",
			targetNamespace: "my-namespace",
			exporters: []generated.Exporter{
				&mockExporter{objects: []client.Object{newConfigMap()}},
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "kind: ConfigMap")
				assert.Contains(t, output, "name: test-config")
				assert.Contains(t, output, "namespace: my-namespace")
				assert.Contains(t, output, "key: value")
			},
		},
		{
			name:            "success with empty exporters",
			targetNamespace: "test",
			exporters:       []generated.Exporter{},
			wantErr:         false,
			validate: func(t *testing.T, output string) {
				// Should only contain the initial separator
				assert.True(t, strings.HasPrefix(output, "---"))
				assert.NotContains(t, output, "kind:")
			},
		},
		{
			name:            "success with exporter returning no objects",
			targetNamespace: "test",
			exporters: []generated.Exporter{
				&mockExporter{objects: []client.Object{}},
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				assert.True(t, strings.HasPrefix(output, "---"))
				assert.NotContains(t, output, "kind:")
			},
		},
		{
			name:            "success with multiple exporters",
			targetNamespace: "test",
			exporters: []generated.Exporter{
				&mockExporter{objects: []client.Object{newConfigMap()}},
				&mockExporter{objects: []client.Object{newConfigMap()}},
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				// Should have two ConfigMaps
				count := strings.Count(output, "kind: ConfigMap")
				assert.Equal(t, 2, count)
			},
		},
		{
			name:            "error from exporter",
			targetNamespace: "test",
			exporters: []generated.Exporter{
				&mockExporter{err: errors.New("export failed")},
			},
			wantErr: true,
			errMsg:  "failed to export resources",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exp := NewGeneratedExporter(GeneratedExporterConfig{
				TargetNamespace: tc.targetNamespace,
				Scheme:          scheme,
				Exporters:       tc.exporters,
			})

			output, err := exp.Run()

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
				assert.Empty(t, output)
			} else {
				require.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, output)
				}
			}
		})
	}
}

func TestGeneratedExporter_IndependentResources(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	newConfigMap := func(name string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
	}

	t.Run("when independentResources is true, exporters receive empty slice", func(t *testing.T) {
		exporter1 := &mockExporter{objects: []client.Object{newConfigMap("config1")}}
		exporter2 := &mockExporter{objects: []client.Object{newConfigMap("config2")}}

		exp := NewGeneratedExporter(GeneratedExporterConfig{
			TargetNamespace:      "test",
			Scheme:               scheme,
			Exporters:            []generated.Exporter{exporter1, exporter2},
			IndependentResources: true,
		})

		_, err := exp.Run()
		require.NoError(t, err)

		// First exporter should receive nil (no previous objects)
		assert.Nil(t, exporter1.receivedRefObjects)
		// Second exporter should also receive nil (independentResources is true means no cross-references)
		assert.Nil(t, exporter2.receivedRefObjects)
	})

	t.Run("when independentResources is false, exporters receive previously exported objects", func(t *testing.T) {
		config1 := newConfigMap("config1")
		exporter1 := &mockExporter{objects: []client.Object{config1}}
		exporter2 := &mockExporter{objects: []client.Object{newConfigMap("config2")}}

		exp := NewGeneratedExporter(GeneratedExporterConfig{
			TargetNamespace: "test",
			Scheme:          scheme,
			Exporters:       []generated.Exporter{exporter1, exporter2},
		})

		_, err := exp.Run()
		require.NoError(t, err)

		// First exporter should receive empty slice (no previous objects yet)
		assert.Empty(t, exporter1.receivedRefObjects)
		// Second exporter should receive the objects from first exporter
		require.Len(t, exporter2.receivedRefObjects, 1)
		assert.Equal(t, "config1", exporter2.receivedRefObjects[0].GetName())
	})
}

func TestGeneratedExporter_ShouldIncludeSecrets(t *testing.T) {
	scheme := runtime.NewScheme()

	tests := []struct {
		name                 string
		independentResources bool
		includeSecrets       bool
		expected             bool
	}{
		{
			name:                 "includeSecrets true, independentResources false",
			independentResources: false,
			includeSecrets:       true,
			expected:             true,
		},
		{
			name:                 "includeSecrets false, independentResources true",
			independentResources: true,
			includeSecrets:       false,
			expected:             true,
		},
		{
			name:                 "both true",
			independentResources: true,
			includeSecrets:       true,
			expected:             true,
		},
		{
			name:                 "both false",
			independentResources: false,
			includeSecrets:       false,
			expected:             false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exp := NewGeneratedExporter(GeneratedExporterConfig{
				TargetNamespace:      "test",
				Scheme:               scheme,
				IndependentResources: tc.independentResources,
				IncludeSecrets:       tc.includeSecrets,
			})
			assert.Equal(t, tc.expected, exp.ShouldIncludeSecrets())
		})
	}
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

func TestGeneratedExporter_BuildCredentialsSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	t.Run("builds secret with credentials when includeSecrets is true", func(t *testing.T) {
		exp := NewGeneratedExporter(GeneratedExporterConfig{
			TargetNamespace: "test-ns",
			Scheme:          scheme,
			IncludeSecrets:  true,
			OrgID:           "org-123",
			CredentialsProvider: &mockCredentialsProvider{
				publicKey:  "pub-key",
				privateKey: "priv-key",
			},
		})

		secret := exp.buildCredentialsSecret()

		require.NotNil(t, secret)
		assert.Equal(t, "atlas-credentials", secret.Name)
		assert.Equal(t, "test-ns", secret.Namespace)
		assert.Equal(t, "org-123", string(secret.Data["orgId"]))
		assert.Equal(t, "pub-key", string(secret.Data["publicApiKey"]))
		assert.Equal(t, "priv-key", string(secret.Data["privateApiKey"]))
	})

	t.Run("builds secret with empty placeholders when includeSecrets is false", func(t *testing.T) {
		exp := NewGeneratedExporter(GeneratedExporterConfig{
			TargetNamespace:      "test-ns",
			Scheme:               scheme,
			IndependentResources: true, // This triggers ShouldIncludeSecrets but with empty creds
			IncludeSecrets:       false,
			OrgID:                "org-123",
		})

		secret := exp.buildCredentialsSecret()

		require.NotNil(t, secret)
		assert.Equal(t, "atlas-credentials", secret.Name)
		assert.Equal(t, "", string(secret.Data["orgId"]))
		assert.Equal(t, "", string(secret.Data["publicApiKey"]))
		assert.Equal(t, "", string(secret.Data["privateApiKey"]))
	})
}

func TestSetConnectionSecretRef(t *testing.T) {
	t.Run("sets connection secret ref on object with ConnectionSecretRef field", func(t *testing.T) {
		obj := &mockCRDWithConnectionSecretRef{
			Spec: mockSpecWithConnectionSecretRef{},
		}

		setConnectionSecretRef(obj, "my-secret")

		require.NotNil(t, obj.Spec.ConnectionSecretRef)
		assert.Equal(t, "my-secret", obj.Spec.ConnectionSecretRef.Name)
	})

	t.Run("does nothing for objects without ConnectionSecretRef field", func(t *testing.T) {
		obj := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
		}

		// Should not panic
		setConnectionSecretRef(obj, "my-secret")
	})
}

// mockCRDWithConnectionSecretRef is a mock CRD object with ConnectionSecretRef for testing.
type mockCRDWithConnectionSecretRef struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              mockSpecWithConnectionSecretRef `json:"spec,omitempty"`
}

type mockSpecWithConnectionSecretRef struct {
	ConnectionSecretRef *k8s.LocalReference `json:"connectionSecretRef,omitempty"`
}

func (m *mockCRDWithConnectionSecretRef) DeepCopyObject() runtime.Object {
	return m
}

// mockCRDObject is a mock Kubernetes object with a Spec field for testing name generation.
type mockCRDObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              mockCRDSpec `json:"spec,omitempty"`
}

type mockCRDSpec struct {
	V20250312 *mockVersionedSpec `json:"v20250312,omitempty"`
}

type mockVersionedSpec struct {
	GroupId *string        `json:"groupId,omitempty"`
	Entry   *mockEntrySpec `json:"entry,omitempty"`
	Name    *string        `json:"name,omitempty"`
}

type mockEntrySpec struct {
	Name     *string `json:"name,omitempty"`
	Username *string `json:"username,omitempty"`
	GroupId  *string `json:"groupId,omitempty"`
}

func (m *mockCRDObject) DeepCopyObject() runtime.Object {
	return m
}

func (m *mockCRDObject) GetObjectKind() schema.ObjectKind {
	return m
}

func strPtr(s string) *string {
	return &s
}

func TestSetResourceName(t *testing.T) {
	tests := []struct {
		name         string
		obj          client.Object
		expectedName string
	}{
		{
			name: "object with name in entry",
			obj: &mockCRDObject{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Cluster",
					APIVersion: "atlas.generated.mongodb.com/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "original-name",
				},
				Spec: mockCRDSpec{
					V20250312: &mockVersionedSpec{
						GroupId: strPtr("project123"),
						Entry: &mockEntrySpec{
							Name: strPtr("my-cluster"),
						},
					},
				},
			},
			expectedName: "cluster-my-cluster",
		},
		{
			name: "object with username for database user",
			obj: &mockCRDObject{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DatabaseUser",
					APIVersion: "atlas.generated.mongodb.com/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "original-name",
				},
				Spec: mockCRDSpec{
					V20250312: &mockVersionedSpec{
						GroupId: strPtr("project456"),
						Entry: &mockEntrySpec{
							Username: strPtr("admin@example.com"),
						},
					},
				},
			},
			expectedName: "databaseuser-adminatexampledotcom",
		},
		{
			name: "object with name at versioned spec level",
			obj: &mockCRDObject{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Group",
					APIVersion: "atlas.generated.mongodb.com/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "original-name",
				},
				Spec: mockCRDSpec{
					V20250312: &mockVersionedSpec{
						Name: strPtr("my-project"),
					},
				},
			},
			expectedName: "group-my-project",
		},
		{
			name: "object without spec preserves original name",
			obj: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-config",
				},
			},
			expectedName: "my-config",
		},
		{
			name: "object with special characters in name gets normalized",
			obj: &mockCRDObject{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Cluster",
					APIVersion: "atlas.generated.mongodb.com/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "original",
				},
				Spec: mockCRDSpec{
					V20250312: &mockVersionedSpec{
						GroupId: strPtr("proj_123"),
						Entry: &mockEntrySpec{
							Name: strPtr("My Cluster (test)"),
						},
					},
				},
			},
			// Special characters are replaced: space -> "-", "(" -> "left-parenthesis", ")" -> "right-parenthesis"
			expectedName: "cluster-my-cluster-left-parenthesistestright-parenthesis",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setResourceName(tc.obj)
			assert.Equal(t, tc.expectedName, tc.obj.GetName())
		})
	}
}

func TestExtractIdentifiers(t *testing.T) {
	tests := []struct {
		name        string
		obj         client.Object
		expectedIDs []string
	}{
		{
			name: "extracts name from entry",
			obj: &mockCRDObject{
				Spec: mockCRDSpec{
					V20250312: &mockVersionedSpec{
						GroupId: strPtr("project123"),
						Entry: &mockEntrySpec{
							Name: strPtr("cluster-name"),
						},
					},
				},
			},
			expectedIDs: []string{"cluster-name"},
		},
		{
			name: "extracts username when name not available",
			obj: &mockCRDObject{
				Spec: mockCRDSpec{
					V20250312: &mockVersionedSpec{
						GroupId: strPtr("project456"),
						Entry: &mockEntrySpec{
							Username: strPtr("testuser"),
						},
					},
				},
			},
			expectedIDs: []string{"testuser"},
		},
		{
			name: "extracts name from versioned spec when entry not available",
			obj: &mockCRDObject{
				Spec: mockCRDSpec{
					V20250312: &mockVersionedSpec{
						Name: strPtr("project-name"),
					},
				},
			},
			expectedIDs: []string{"project-name"},
		},
		{
			name: "returns nil for object without spec",
			obj: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "config",
				},
			},
			expectedIDs: nil,
		},
		{
			name: "extracts name from entry ignoring groupId",
			obj: &mockCRDObject{
				Spec: mockCRDSpec{
					V20250312: &mockVersionedSpec{
						Entry: &mockEntrySpec{
							GroupId: strPtr("entry-group"),
							Name:    strPtr("resource-name"),
						},
					},
				},
			},
			expectedIDs: []string{"resource-name"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ids := extractIdentifiers(tc.obj)
			assert.Equal(t, tc.expectedIDs, ids)
		})
	}
}

func TestGetStringField(t *testing.T) {
	type testStruct struct {
		DirectString  string
		PointerString *string
		IntField      int
	}

	strValue := "pointer-value"

	tests := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "direct string field",
			fieldName: "DirectString",
			expected:  "direct-value",
		},
		{
			name:      "pointer string field",
			fieldName: "PointerString",
			expected:  "pointer-value",
		},
		{
			name:      "non-existent field",
			fieldName: "NonExistent",
			expected:  "",
		},
		{
			name:      "non-string field",
			fieldName: "IntField",
			expected:  "",
		},
	}

	obj := testStruct{
		DirectString:  "direct-value",
		PointerString: &strValue,
		IntField:      42,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getStringField(reflect.ValueOf(obj), tc.fieldName)
			assert.Equal(t, tc.expected, result)
		})
	}
}
