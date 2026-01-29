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
	ProfileAtlasCLI                       = "Name of the profile to use from your configuration file. To learn about profiles for the Atlas CLI, see https://dochub.mongodb.org/core/atlas-cli-save-connection-settings."
	ProjectID                             = "Hexadecimal string that identifies the project to use. This option overrides the settings in the configuration file or environment variable."
	OrgID                                 = "Organization ID to use. This option overrides the settings in the configuration file or environment variable."
	ExporterClusterName                   = "One or more comma separated cluster names to import"
	Debug                                 = "Debug log level."
	OperatorIncludeSecrets                = "Flag that generates kubernetes secrets with data for projects, users, deployments entities."
	OperatorTargetNamespace               = "Namespaces to use for generated kubernetes entities"
	OperatorVersion                       = "Version of Atlas Kubernetes Operator to generate resources for."
	OperatorVersionInstall                = "Version of the operator to install."
	OperatorTargetNamespaceInstall        = "Namespace where to install the operator."
	OperatorWatchNamespace                = "List that contains namespaces that the operator will listen to."
	OperatorProjectName                   = "Name of the project to create or use with the installed operator."
	OperatorImport                        = "Flag that indicates whether to import existing Atlas resources into the cluster for the operator to manage."
	OperatorAtlasGov                      = "Flag that indicates whether to configure Atlas for Government as a target of the operator."
	KubernetesClusterConfig               = "Path to the kubeconfig file to use for CLI requests."
	KubernetesClusterContext              = "Name of the kubeconfig context to use."
	OperatorResourceDeletionProtection    = "Toggle atlas operator deletion protection for resources like Projects, Deployments, etc. Read more: https://dochub.mongodb.org/core/ako-deletion-protection"
	OperatorSubResourceDeletionProtection = "Toggle atlas operator deletion protection for subresources like Alerts, Integrations, etc. Read more: https://dochub.mongodb.org/core/ako-deletion-protection"
	OperatorConfigOnly                    = "Flag that indicates whether to generate only the operator configuration files without installing the Operator"
	ExporterDataFederationName            = "One or more comma separated data federation names to import"
	IndependentResources                  = "Flag that makes the generated resources that support independent usage, to use external IDs rather than Kubernetes references."
	EnableWatch                           = "Flag that indicates whether to watch the command until it completes its execution or the watch times out. To set the time that the watch times out, use the --watchTimeout option."
	WatchTimeout                          = "Time in seconds until a watch times out. After a watch times out, the CLI no longer watches the command."
	IPAccessList                          = "A comma-separated list of IP or CIDR block to allowlist for Operator to communicate with Atlas APIs. Read more: https://www.mongodb.com/docs/atlas/configure-api-access-project/"
	CRDVersion                            = "Version of the CRD to generate. Valid values are 'curated' or 'generated'."
)
