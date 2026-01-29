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

// Package exporter provides public exporter implementations for Atlas resources.
// The GeneratedExporter handles resources built using the automated generation mechanism
// with the Crapi library for CRD-to-SDK type translation.
package exporter

import (
	"errors"
)

// ErrNotImplemented is returned when the generated exporter is called but not yet implemented.
var ErrNotImplemented = errors.New("generated exporter not implemented yet")

// GeneratedExporter handles the export of Atlas resources using auto-generated CRDs.
// It leverages the Crapi library for bi-directional translation between Go types
// representing CRDs and the corresponding Go types within the Atlas SDK.
type GeneratedExporter struct {
	// Configuration fields will be added as the implementation progresses.
	// Expected fields include:
	// - Crapi translator instance
	// - SDK client for Atlas API calls
	// - Target namespace and other export options
}

// NewGeneratedExporter creates a new instance of GeneratedExporter.
// Additional configuration options will be added as the implementation progresses.
func NewGeneratedExporter() *GeneratedExporter {
	return &GeneratedExporter{}
}

// Run executes the generated resource export workflow.
// This is a placeholder implementation that will be expanded to:
// 1. Build and initialize the Crapi translator
// 2. Retrieve resources from MongoDB Atlas using the SDK
// 3. Delegate type conversion to the Crapi translator
// 4. Serialize the CRD objects to YAML format
func (e *GeneratedExporter) Run() (string, error) {
	return "", ErrNotImplemented
}
