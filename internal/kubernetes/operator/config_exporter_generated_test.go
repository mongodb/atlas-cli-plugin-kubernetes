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

	generated "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/exporter/generated"
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

	exp := NewGeneratedExporter("test-namespace", scheme, exporters, true)

	require.NotNil(t, exp)
	assert.Equal(t, "test-namespace", exp.targetNamespace)
	assert.Equal(t, scheme, exp.scheme)
	assert.Len(t, exp.exporters, 1)
	assert.True(t, exp.independentResources)
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
			exp := NewGeneratedExporter(tc.targetNamespace, scheme, tc.exporters, false)

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

		exp := NewGeneratedExporter("test", scheme, []generated.Exporter{exporter1, exporter2}, true)

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

		exp := NewGeneratedExporter("test", scheme, []generated.Exporter{exporter1, exporter2}, false)

		_, err := exp.Run()
		require.NoError(t, err)

		// First exporter should receive empty slice (no previous objects yet)
		assert.Empty(t, exporter1.receivedRefObjects)
		// Second exporter should receive the objects from first exporter
		require.Len(t, exporter2.receivedRefObjects, 1)
		assert.Equal(t, "config1", exporter2.receivedRefObjects[0].GetName())
	})
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
