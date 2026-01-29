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

package operator

// Exporter defines the interface for exporting Atlas resources to Kubernetes manifests.
// This interface allows switching between different exporter implementations based on
// the CRD version (curated vs generated).
type Exporter interface {
	// Run executes the export process and returns the serialized Kubernetes manifests
	// as a YAML string, or an error if the export fails.
	Run() (string, error)
}
