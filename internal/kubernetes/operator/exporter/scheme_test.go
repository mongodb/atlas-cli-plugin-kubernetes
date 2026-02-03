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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewScheme(t *testing.T) {
	scheme, err := NewScheme()
	require.NoError(t, err)
	require.NotNil(t, scheme)

	tests := []struct {
		kind       string
		registered bool
	}{
		{"Group", true},
		{"Cluster", true},
		{"FlexCluster", true},
		{"DatabaseUser", true},
		{"UnknownType", false},
	}

	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			registered := scheme.Recognizes(gvk(tc.kind))
			assert.Equal(t, tc.registered, registered)
		})
	}
}

func TestNewScheme_AddToSchemeError(t *testing.T) {
	// Save original and restore after test
	originalFunc := addToSchemeFunc
	defer func() { addToSchemeFunc = originalFunc }()

	addToSchemeFunc = func(_ *runtime.Scheme) error {
		return errors.New("add to scheme error")
	}

	scheme, err := NewScheme()

	require.Error(t, err)
	assert.Nil(t, scheme)
	assert.Contains(t, err.Error(), "add to scheme error")
}

func gvk(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "atlas.generated.mongodb.com",
		Version: "v1",
		Kind:    kind,
	}
}
