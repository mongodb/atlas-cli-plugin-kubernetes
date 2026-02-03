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
	generated "github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/exporter/generated"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/crapi"
	admin "go.mongodb.org/atlas-sdk/v20250312013/admin"
)

// ExporterFactory creates an exporter for a specific resource type.
type ExporterFactory func(client *admin.APIClient, translator crapi.Translator, identifiers []string) generated.Exporter

// ResourceConfig defines a resource type that can be exported.
// To add a new resource, add an entry to the SupportedResources slice.
type ResourceConfig struct {
	// CRDName is the full CRD name (e.g., "groups.atlas.generated.mongodb.com")
	CRDName string

	// Factory creates the exporter for this resource type
	Factory ExporterFactory
}

// SupportedResources lists all resources that can be exported.
// To add a new resource, simply add an entry here.
var SupportedResources = []ResourceConfig{
	{CRDName: "groups.atlas.generated.mongodb.com", Factory: newGroupExporter},
	{CRDName: "clusters.atlas.generated.mongodb.com", Factory: newClusterExporter},
	{CRDName: "flexclusters.atlas.generated.mongodb.com", Factory: newFlexClusterExporter},
	{CRDName: "databaseusers.atlas.generated.mongodb.com", Factory: newDatabaseUserExporter},
}

// Factory wrapper functions to match the ExporterFactory signature

func newGroupExporter(client *admin.APIClient, translator crapi.Translator, identifiers []string) generated.Exporter {
	return generated.NewGroupExporter(client, translator, identifiers)
}

func newClusterExporter(client *admin.APIClient, translator crapi.Translator, identifiers []string) generated.Exporter {
	return generated.NewClusterExporter(client, translator, identifiers)
}

func newFlexClusterExporter(client *admin.APIClient, translator crapi.Translator, identifiers []string) generated.Exporter {
	return generated.NewFlexClusterExporter(client, translator, identifiers)
}

func newDatabaseUserExporter(client *admin.APIClient, translator crapi.Translator, identifiers []string) generated.Exporter {
	return generated.NewDatabaseUserExporter(client, translator, identifiers)
}
