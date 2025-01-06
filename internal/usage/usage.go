// Copyright 2021 MongoDB Inc
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

package usage

const (
	ProjectID                  = "Hexadecimal string that identifies the project to use. This option overrides the settings in the configuration file or environment variable."
	OrgID                      = "Organization ID to use. This option overrides the settings in the configuration file or environment variable."
	ExporterClusterName        = "One or more comma separated cluster names to import"
	OperatorIncludeSecrets     = "Flag that generates kubernetes secrets with data for projects, users, deployments entities."
	OperatorTargetNamespace    = "Namespaces to use for generated kubernetes entities"
	OperatorVersion            = "Version of Atlas Kubernetes Operator to generate resources for."
	ExporterDataFederationName = "One or more comma separated data federation names to import"
	IndependentResources       = "Flag that makes the generated resources that support independent usage, to use external IDs rather than Kubernetes references."
)
