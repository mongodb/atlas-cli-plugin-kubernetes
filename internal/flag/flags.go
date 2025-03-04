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

package flag

const (
	Profile                               = "profile"                       // Profile flag to use a profile
	ProfileShort                          = "P"                             // ProfileShort flag to use a profile
	OrgID                                 = "orgId"                         // OrgID flag to use an Organization ID
	ProjectID                             = "projectId"                     // ProjectID flag to use a project ID
	ClusterName                           = "clusterName"                   // ClusterName flag
	Debug                                 = "debug"                         // Debug flag to set debug log level
	DebugShort                            = "D"                             // DebugShort flag to set debug log level
	EnableWatch                           = "watch"                         // EnableWatch flag
	EnableWatchShort                      = "w"                             // EnableWatchShort flag
	WatchTimeout                          = "watchTimeout"                  // WatchTimeout flag
	OperatorIncludeSecrets                = "includeSecrets"                // OperatorIncludeSecrets flag
	OperatorTargetNamespace               = "targetNamespace"               // OperatorTargetNamespace flag
	OperatorWatchNamespace                = "watchNamespace"                // OperatorWatchNamespace flag
	OperatorVersion                       = "operatorVersion"               // OperatorVersion flag
	OperatorProjectName                   = "projectName"                   // OperatorProjectName flag
	OperatorImport                        = "import"                        // OperatorImport flag
	OperatorResourceDeletionProtection    = "resourceDeletionProtection"    // OperatorResourceDeletionProtection flag
	OperatorSubResourceDeletionProtection = "subresourceDeletionProtection" // Operator OperatorSubResourceDeletionProtection flag
	OperatorConfigOnly                    = "configOnly"
	OperatorAtlasGov                      = "atlasGov"             // OperatorAtlasGov flag
	KubernetesClusterConfig               = "kubeconfig"           // Kubeconfig flag
	KubernetesClusterContext              = "kubeContext"          // KubeContext flag
	DataFederationName                    = "dataFederationName"   // DataFederationName flag
	IndependentResources                  = "independentResources" // IndependentResources flag
)
