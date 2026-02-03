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

import (
	"bytes"
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	generated "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/exporter/generated"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GeneratedExporter handles the export of Atlas resources using auto-generated CRDs.
// It leverages the Crapi library for bi-directional translation between Go types
// representing CRDs and the corresponding Go types within the Atlas SDK.
type GeneratedExporter struct {
	// targetNamespace is the Kubernetes namespace for the exported resources
	targetNamespace string

	// scheme is the Kubernetes runtime scheme for serialization
	scheme *runtime.Scheme

	// exporters holds the resource-specific exporters
	exporters []generated.Exporter
}

// NewGeneratedExporter creates a new instance of GeneratedExporter.
func NewGeneratedExporter(targetNamespace string, scheme *runtime.Scheme, exporters []generated.Exporter) *GeneratedExporter {
	return &GeneratedExporter{
		targetNamespace: targetNamespace,
		scheme:          scheme,
		exporters:       exporters,
	}
}

// Run executes the generated resource export workflow.
// It retrieves resources from MongoDB Atlas using the SDK,
// delegates type conversion to the Crapi translator,
// and serializes the CRD objects to YAML format.
func (e *GeneratedExporter) Run() (string, error) {
	ctx := context.Background()
	output := bytes.NewBufferString(yamlSeparator)

	serializer := json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		e.scheme,
		e.scheme,
		json.SerializerOptions{Yaml: true, Pretty: true},
	)

	// Export all resources from all exporters
	for _, exp := range e.exporters {
		objects, err := exp.Export(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to export resources: %w", err)
		}

		for _, obj := range objects {
			// Set the target namespace if specified
			if e.targetNamespace != "" {
				if clientObj, ok := obj.(client.Object); ok {
					clientObj.SetNamespace(e.targetNamespace)
				}
			}

			if err := serializer.Encode(obj, output); err != nil {
				return "", fmt.Errorf("failed to serialize resource: %w", err)
			}
			output.WriteString(yamlSeparator)
		}
	}

	return output.String(), nil
}
