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
	"reflect"
	"strings"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/resources"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	generated "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/exporter/generated"
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
	var exportedObjects []client.Object
	for _, exp := range e.exporters {
		objects, err := exp.Export(ctx, exportedObjects)
		if err != nil {
			return "", fmt.Errorf("failed to export resources: %w", err)
		}

		for _, obj := range objects {
			// Set hierarchical Kubernetes-compliant name
			setResourceName(obj)

			// Set the target namespace if specified
			if e.targetNamespace != "" {
				obj.SetNamespace(e.targetNamespace)
			}

			if err := serializer.Encode(obj, output); err != nil {
				return "", fmt.Errorf("failed to serialize resource: %w", err)
			}
			output.WriteString(yamlSeparator)
		}

		exportedObjects = append(exportedObjects, objects...)
	}

	return output.String(), nil
}

// setResourceName sets a Kubernetes-compliant name on the object.
// The name is derived from the resource kind and the resource name from the spec.
// Format: {kind}-{name}
// If no name can be extracted, the existing name is preserved.
func setResourceName(obj client.Object) {
	kind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
	if kind == "" {
		return
	}

	// Extract identifying information from the spec
	identifiers := extractIdentifiers(obj)

	// If no identifiers found, preserve the existing name
	if len(identifiers) == 0 {
		return
	}

	// Build hierarchical name: kind-identifier1-identifier2...
	name := kind + "-" + strings.Join(identifiers, "-")

	// Normalize to be Kubernetes DNS-1123 compliant
	name = resources.NormalizeAtlasName(name, resources.AtlasNameToKubernetesName())
	obj.SetName(name)
}

// extractIdentifiers extracts identifying fields from the object's spec.
// It uses reflection to navigate the versioned spec structure and extract
// the resource name (Name or Username fields).
func extractIdentifiers(obj client.Object) []string {
	// Use reflection to access the Spec field
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	specField := objValue.FieldByName("Spec")
	if !specField.IsValid() {
		return nil
	}

	// Navigate through versioned spec structure (e.g., V20250312)
	// The generated types have a nested structure: Spec.V{version}.Entry.{field}
	for i := 0; i < specField.NumField(); i++ {
		versionField := specField.Field(i)
		if versionField.Kind() == reflect.Ptr && !versionField.IsNil() {
			versionField = versionField.Elem()
			if name := extractNameFromVersionedSpec(versionField); name != "" {
				return []string{name}
			}
			break
		}
	}

	return nil
}

// extractNameFromVersionedSpec extracts the resource name from the versioned spec struct.
// It looks for Name or Username fields in the Entry struct or directly on the spec.
func extractNameFromVersionedSpec(v reflect.Value) string {
	// Check Entry field first (most generated types have this)
	entryField := v.FieldByName("Entry")
	if entryField.IsValid() && entryField.Kind() == reflect.Ptr && !entryField.IsNil() {
		entryValue := entryField.Elem()
		if name := getStringField(entryValue, "Name"); name != "" {
			return name
		}
		if username := getStringField(entryValue, "Username"); username != "" {
			return username
		}
	}

	// Fallback: check directly on the versioned spec
	if name := getStringField(v, "Name"); name != "" {
		return name
	}
	if username := getStringField(v, "Username"); username != "" {
		return username
	}

	return ""
}

// getStringField extracts a string value from a struct field by name.
// It handles both direct string fields and pointer to string fields.
func getStringField(v reflect.Value, fieldName string) string {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}

	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Ptr:
		if !field.IsNil() && field.Elem().Kind() == reflect.String {
			return field.Elem().String()
		}
	}
	return ""
}
