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

//go:build e2e || generate

package e2e

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/crd2go/crd2go/k8s"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/features"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/resources"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/pointer"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/test"
	akov2generated "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/crapi/testdata/samples/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// parseYAMLDocuments parses a multi-document YAML string into a slice of client.Object.
func parseYAMLDocuments(t *testing.T, yamlContent string) []client.Object {
	t.Helper()
	var objects []client.Object

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(yamlContent)), 4096)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("failed to decode YAML document: %v", err)
		}
		if len(obj.Object) > 0 {
			objects = append(objects, obj)
		}
	}
	return objects
}

// findByKind returns the first object with the specified kind.
func findByKind(objects []client.Object, kind string) client.Object {
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Kind == kind {
			return obj
		}
	}
	return nil
}

// convertTo converts a client.Object to the specified typed object.
func convertTo[T any](t *testing.T, obj client.Object) *T {
	t.Helper()
	u := obj.(*unstructured.Unstructured)
	result := new(T)
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, result)
	require.NoError(t, err)
	return result
}

// resourceName builds the expected Kubernetes resource name from kind and identifier.
func resourceName(kind, identifier string) string {
	name := strings.ToLower(kind) + "-" + identifier
	return resources.NormalizeAtlasName(name, resources.AtlasNameToKubernetesName())
}

// TestGeneratedExporterWithResources tests the generated CRD exporter with real Atlas resources.
func TestGeneratedExporterWithResources(t *testing.T) {
	orgId := os.Getenv("MCLI_ORG_ID")

	s := InitialSetup(t)
	cliPath := s.cliPath
	generator := s.generator

	// Create shared resources in Atlas
	generator.tier = "M0"
	generator.generateCluster()
	generator.generateFlexCluster()
	generator.generateDBUser("test-db-user-")

	tests := []struct {
		name                 string
		independentResources bool
		includeSecrets       bool
		expectedCount        int
		expectSecret         bool
	}{
		{
			name:                 "export without flags",
			independentResources: false,
			includeSecrets:       false,
			expectedCount:        4,
			expectSecret:         false,
		},
		{
			name:                 "export with independentResources",
			independentResources: true,
			includeSecrets:       false,
			expectedCount:        5,
			expectSecret:         true,
		},
		{
			name:                 "export with includeSecrets",
			independentResources: false,
			includeSecrets:       true,
			expectedCount:        5,
			expectSecret:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmdArgs := []string{
				"kubernetes", "config", "generate",
				"--projectId", generator.projectID,
				"--crdVersion", features.CRDVersionGenerated,
				"--targetNamespace", targetNamespace,
			}
			if tc.independentResources {
				cmdArgs = append(cmdArgs, "--independentResources")
			}
			if tc.includeSecrets {
				cmdArgs = append(cmdArgs, "--includeSecrets")
			}

			t.Logf("executing: %s", strings.Join(cmdArgs, " "))
			cmd := exec.Command(cliPath, cmdArgs...)
			cmd.Env = os.Environ()
			resp, err := test.RunAndGetStdOut(cmd)
			t.Log(string(resp))
			require.NoError(t, err, string(resp))

			objects := parseYAMLDocuments(t, string(resp))
			assert.Equal(t, tc.expectedCount, len(objects))

			// Validate Secret
			if tc.expectSecret {
				secret := convertTo[corev1.Secret](t, findByKind(objects, "Secret"))
				assert.Equal(t, "Secret", objects[0].GetObjectKind().GroupVersionKind().Kind)
				assert.Equal(t, metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}, secret.TypeMeta)
				assert.Equal(t, "atlas-credentials", secret.Name)
				assert.Equal(t, targetNamespace, secret.Namespace)
				assert.Equal(t, "credentials", secret.Labels["atlas.mongodb.com/type"])
				assert.Contains(t, secret.Data, "orgId")
				assert.Contains(t, secret.Data, "publicApiKey")
				assert.Contains(t, secret.Data, "privateApiKey")
			} else {
				assert.Nil(t, findByKind(objects, "Secret"))
			}

			groupName := resourceName("Group", generator.projectName)

			// Validate Group
			group := convertTo[akov2generated.Group](t, findByKind(objects, "Group"))
			assert.Equal(t, expectedGroup(orgId, generator.projectName, tc.expectSecret), group)

			// Validate Cluster
			cluster := convertTo[akov2generated.Cluster](t, findByKind(objects, "Cluster"))
			assert.Equal(t, expectedCluster(cluster, groupName, generator.projectID, generator.clusterName, tc.independentResources, tc.expectSecret), cluster)

			// Validate FlexCluster
			flex := convertTo[akov2generated.FlexCluster](t, findByKind(objects, "FlexCluster"))
			assert.Equal(t, expectedFlexCluster(groupName, generator.projectID, generator.flexName, tc.independentResources, tc.expectSecret), flex)

			// Validate DatabaseUser
			dbUser := convertTo[akov2generated.DatabaseUser](t, findByKind(objects, "DatabaseUser"))
			assert.Equal(t, expectedDatabaseUser(groupName, generator.projectID, generator.dbUser, tc.independentResources, tc.expectSecret), dbUser)
		})
	}
}

func connectionSecretRef(expectSecret bool) *k8s.LocalReference {
	if expectSecret {
		return &k8s.LocalReference{Name: "atlas-credentials"}
	}
	return nil
}

func expectedGroup(orgId, name string, expectSecret bool) *akov2generated.Group {
	return &akov2generated.Group{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "atlas.generated.mongodb.com/v1",
			Kind:       "Group",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName("Group", name),
			Namespace: targetNamespace,
		},
		Spec: akov2generated.GroupSpec{
			ConnectionSecretRef: connectionSecretRef(expectSecret),
			V20250312: &akov2generated.GroupSpecV20250312{
				Entry: &akov2generated.GroupSpecV20250312Entry{
					Name:                      name,
					OrgId:                     orgId,
					Tags:                      &[]akov2generated.Tags{},
					WithDefaultAlertsSettings: pointer.Get(false),
				},
			},
		},
	}
}

func expectedCluster(actual *akov2generated.Cluster, groupName, groupID, clusterName string, independentResources, expectSecret bool) *akov2generated.Cluster {
	// ZoneId is assigned by Atlas and not predictable - copy from actual
	var zoneId *string
	var regionName *string
	if actual.Spec.V20250312 != nil && actual.Spec.V20250312.Entry != nil && len(*actual.Spec.V20250312.Entry.ReplicationSpecs) > 0 {
		zoneId = (*actual.Spec.V20250312.Entry.ReplicationSpecs)[0].ZoneId
		replicationSpec := (*actual.Spec.V20250312.Entry.ReplicationSpecs)[0]
		if len(*replicationSpec.RegionConfigs) > 0 {
			regionName = (*replicationSpec.RegionConfigs)[0].RegionName
		}
	}

	spec := akov2generated.ClusterSpecV20250312{
		Entry: &akov2generated.ClusterSpecV20250312Entry{
			Name:        pointer.Get(clusterName),
			ClusterType: pointer.Get("REPLICASET"),
			ReplicationSpecs: &[]akov2generated.ReplicationSpecs{
				{
					ZoneId:   zoneId,
					ZoneName: pointer.Get("Zone 1"),
					RegionConfigs: &[]akov2generated.RegionConfigs{
						{
							ProviderName:        pointer.Get("TENANT"),
							BackingProviderName: pointer.Get("AWS"),
							RegionName:          regionName,
							Priority:            pointer.Get(7),
							ElectableSpecs: &akov2generated.ElectableSpecs{
								InstanceSize:          pointer.Get("M0"),
								EffectiveInstanceSize: pointer.Get("M0"),
								DiskSizeGB:            pointer.Get(0.5),
							},
						},
					},
				},
			},
			BackupEnabled: pointer.Get(false),
			BiConnector: &akov2generated.BiConnector{
				Enabled:        pointer.Get(false),
				ReadPreference: pointer.Get("secondary"),
			},
			DiskWarmingMode:                  pointer.Get("FULLY_WARMED"),
			EncryptionAtRestProvider:         pointer.Get("NONE"),
			GlobalClusterSelfManagedSharding: pointer.Get(false),
			MongoDBMajorVersion:              pointer.Get("8.0"),
			Paused:                           pointer.Get(false),
			PitEnabled:                       pointer.Get(false),
			RedactClientLogData:              pointer.Get(false),
			RetainBackups:                    pointer.Get(false),
			RootCertType:                     pointer.Get("ISRGROOTX1"),
			TerminationProtectionEnabled:     pointer.Get(false),
			UseAwsTimeBasedSnapshotCopyForFastInitialSync: pointer.Get(false),
			VersionReleaseSystem:                          pointer.Get("LTS"),
			Tags:                                          &[]akov2generated.Tags{},
			Labels:                                        &[]akov2generated.Tags{},
		},
	}

	if independentResources {
		spec.GroupId = pointer.Get(groupID)
		spec.GroupRef = &k8s.LocalReference{Name: ""}
	} else {
		spec.GroupRef = &k8s.LocalReference{Name: groupName}
	}

	return &akov2generated.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "atlas.generated.mongodb.com/v1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName("Cluster", clusterName),
			Namespace: targetNamespace,
		},
		Spec: akov2generated.ClusterSpec{
			ConnectionSecretRef: connectionSecretRef(expectSecret),
			V20250312:           &spec,
		},
	}
}

func expectedFlexCluster(groupName, groupID, flexName string, independentResources, expectSecret bool) *akov2generated.FlexCluster {
	spec := akov2generated.FlexClusterSpecV20250312{
		Entry: &akov2generated.FlexClusterSpecV20250312Entry{
			Name: flexName,
			ProviderSettings: akov2generated.ProviderSettings{
				BackingProviderName: "AWS",
				RegionName:          "US_EAST_1",
			},
			Tags:                         &[]akov2generated.Tags{},
			TerminationProtectionEnabled: pointer.Get(false),
		},
	}

	if independentResources {
		spec.GroupId = pointer.Get(groupID)
	} else {
		spec.GroupRef = &k8s.LocalReference{Name: groupName}
	}

	return &akov2generated.FlexCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "atlas.generated.mongodb.com/v1",
			Kind:       "FlexCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName("FlexCluster", flexName),
			Namespace: targetNamespace,
		},
		Spec: akov2generated.FlexClusterSpec{
			ConnectionSecretRef: connectionSecretRef(expectSecret),
			V20250312:           &spec,
		},
	}
}

func expectedDatabaseUser(groupName, groupID, username string, independentResources, expectSecret bool) *akov2generated.DatabaseUser {
	spec := akov2generated.DatabaseUserSpecV20250312{
		Entry: &akov2generated.DatabaseUserSpecV20250312Entry{
			Username:     username,
			DatabaseName: "$external",
			AwsIAMType:   pointer.Get("NONE"),
			LdapAuthType: pointer.Get("NONE"),
			OidcAuthType: pointer.Get("NONE"),
			X509Type:     pointer.Get("MANAGED"),
			Roles: &[]akov2generated.Roles{
				{
					DatabaseName: "admin",
					RoleName:     "readAnyDatabase",
				},
			},
			Scopes: &[]akov2generated.Scopes{},
			Labels: &[]akov2generated.Tags{},
		},
	}

	if independentResources {
		spec.GroupId = pointer.Get(groupID)
	} else {
		spec.GroupRef = &k8s.LocalReference{Name: groupName}
	}

	return &akov2generated.DatabaseUser{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "atlas.generated.mongodb.com/v1",
			Kind:       "DatabaseUser",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName("DatabaseUser", username),
			Namespace: targetNamespace,
		},
		Spec: akov2generated.DatabaseUserSpec{
			ConnectionSecretRef: connectionSecretRef(expectSecret),
			V20250312:           &spec,
		},
	}
}
