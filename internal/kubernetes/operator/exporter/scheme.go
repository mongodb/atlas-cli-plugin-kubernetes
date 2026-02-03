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

package exporter

import (
	generatedv1 "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/crapi/testdata/samples/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// addToSchemeFunc is a function variable for dependency injection in tests.
var addToSchemeFunc = generatedv1.AddToScheme

// NewScheme creates a new runtime.Scheme with the generated CRD types registered.
func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	// Register the generated v1 types (Group, Cluster, etc.)
	if err := addToSchemeFunc(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}
