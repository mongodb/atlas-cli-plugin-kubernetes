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
	"strings"
	"testing"

	generated "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/exporter/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// mockExporter is a simple mock implementation of generated.Exporter for testing.
type mockExporter struct {
	objects []client.Object
	err     error
}

func (m *mockExporter) Export(_ context.Context) ([]client.Object, error) {
	return m.objects, m.err
}

func TestNewGeneratedExporter(t *testing.T) {
	scheme := runtime.NewScheme()
	exporters := []generated.Exporter{&mockExporter{}}

	exp := NewGeneratedExporter("test-namespace", scheme, exporters)

	require.NotNil(t, exp)
	assert.Equal(t, "test-namespace", exp.targetNamespace)
	assert.Equal(t, scheme, exp.scheme)
	assert.Len(t, exp.exporters, 1)
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
			exp := NewGeneratedExporter(tc.targetNamespace, scheme, tc.exporters)

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
