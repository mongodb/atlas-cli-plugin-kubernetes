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
	"fmt"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/crds"
	generated "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/exporter/generated"
	admin "go.mongodb.org/atlas-sdk/v20250312013/admin"
	"k8s.io/apimachinery/pkg/runtime"
)

// Function variables for dependency injection in tests
var (
	newSDKClientFunc = NewSDKClient
	newSchemeFunc    = NewScheme
)

// sdkClientCreator is a function type for creating SDK clients.
type sdkClientCreator func(service string) (*admin.APIClient, error)

// schemeCreator is a function type for creating runtime schemes.
type schemeCreator func() (*runtime.Scheme, error)

// SetupConfig holds the configuration for setting up the generated exporter.
type SetupConfig struct {
	// ProjectID is the Atlas project ID to export
	ProjectID string

	// TargetNamespace is the Kubernetes namespace for exported resources
	TargetNamespace string

	// Service is the Atlas service type (e.g., "cloud", "cloudgov")
	Service string

	// CRDProvider provides CRD definitions
	CRDProvider crds.AtlasOperatorCRDProvider

	// OperatorVersion is the operator version for CRD lookup
	OperatorVersion string
}

// Setup creates and configures a GeneratedExporter with all required dependencies.
// It iterates over SupportedResources to create exporters for each resource type.
func Setup(cfg SetupConfig) (*operator.GeneratedExporter, error) {
	// Create the SDK client using CLI configuration
	sdkClient, err := newSDKClientFunc(cfg.Service)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK client: %w", err)
	}

	// Create the scheme once for all translators
	scheme, err := newSchemeFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheme: %w", err)
	}

	exporters := make([]generated.Exporter, 0, len(SupportedResources))

	// Create exporters for all supported resources
	for _, resource := range SupportedResources {
		crd, err := cfg.CRDProvider.GetAtlasOperatorResource(resource.CRDName, cfg.OperatorVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch CRD %s: %w", resource.CRDName, err)
		}

		translator, err := NewTranslator(scheme, crd, SDKVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to create translator for %s: %w", resource.CRDName, err)
		}

		exporter := resource.Factory(sdkClient, translator, []string{cfg.ProjectID})
		exporters = append(exporters, exporter)
	}

	// Create the GeneratedExporter
	return operator.NewGeneratedExporter(cfg.TargetNamespace, scheme, exporters), nil
}
