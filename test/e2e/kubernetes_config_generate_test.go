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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api"
	akoapi "github.com/mongodb/mongodb-atlas-kubernetes/v2/api"
	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1"
	akov2common "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/common"
	akov2project "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/project"
	akov2provider "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/provider"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/status"
	akov2status "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	atlasv2 "go.mongodb.org/atlas-sdk/v20250312006/admin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/features"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/resources"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/secrets"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/pointer"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/test"
)

const (
	targetNamespace = "importer-namespace"
	credSuffixTest  = "-credentials"
	activeStatus    = "ACTIVE"
)

// These kinds represent global types in AKO which are independent of any Atlas Project.
// They can be filtered in concurrent e2e tests if they are not relevant for assertion.
var globalkinds = []string{"AtlasFederatedAuth", "AtlasOrgSettings"}

var (
	federationSettingsID   string
	identityProviderStatus string
	samlIdentityProviderID string
	expectedLabels         = map[string]string{
		features.ResourceVersion: features.LatestOperatorMajorVersion,
	}
)

func getK8SEntities(data []byte) ([]runtime.Object, error) {
	b := bufio.NewReader(bytes.NewReader(data))
	r := yaml.NewYAMLReader(b)

	var result []runtime.Object

	for {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		d := scheme.Codecs.UniversalDeserializer()
		obj, _, err := d.Decode(doc, nil, nil)
		if err != nil {
			// if document is not a K8S object, skip it
			continue
		}
		if obj != nil {
			result = append(result, obj)
		}
	}
	return result, nil
}

type KubernetesConfigGenerateProjectSuite struct {
	generator       *atlasE2ETestGenerator
	expectedProject *akov2.AtlasProject
	cliPath         string
	atlasCliPath    string
}

const projectPrefix = "Kubernetes-"

func InitialSetupWithTeam(t *testing.T) KubernetesConfigGenerateProjectSuite {
	t.Helper()
	s := KubernetesConfigGenerateProjectSuite{}
	s.generator = newAtlasE2ETestGenerator(t)
	s.generator.generateTeam("Kubernetes")

	s.generator.generateEmptyProject(projectPrefix + s.generator.projectName)
	s.expectedProject = referenceProject(s.generator.projectName, targetNamespace, expectedLabels)

	cliPath, err := PluginBin()
	require.NoError(t, err)
	s.cliPath = cliPath

	atlasCliPath, err := AtlasCLIBin()
	require.NoError(t, err)
	s.atlasCliPath = atlasCliPath

	// always register atlas entities
	require.NoError(t, akov2.AddToScheme(scheme.Scheme))
	return s
}

func InitialSetup(t *testing.T) KubernetesConfigGenerateProjectSuite {
	t.Helper()
	s := KubernetesConfigGenerateProjectSuite{}
	s.generator = newAtlasE2ETestGenerator(t)
	s.generator.generateEmptyProject(projectPrefix + s.generator.projectName)
	s.expectedProject = referenceProject(s.generator.projectName, targetNamespace, expectedLabels)

	cliPath, err := PluginBin()
	require.NoError(t, err)
	s.cliPath = cliPath

	atlasCliPath, err := AtlasCLIBin()
	require.NoError(t, err)
	s.atlasCliPath = atlasCliPath

	// always register atlas entities
	require.NoError(t, akov2.AddToScheme(scheme.Scheme))
	return s
}

func TestExportWorksWithoutFedAuth(t *testing.T) {
	s := InitialSetup(t)
	cliPath := s.cliPath
	generator := s.generator
	cmd := exec.Command(cliPath,
		"kubernetes",
		"config",
		"generate",
		"--projectId",
		generator.projectID)
	cmd.Env = os.Environ()
	resp, err := cmd.CombinedOutput()
	t.Log(string(resp))
	require.NoError(t, err, string(resp))
	var objects []runtime.Object
	objects, err = getK8SEntities(resp)
	require.NoError(t, err, "should not fail on decode but got:\n"+string(resp))
	require.NotEmpty(t, objects)
}

func TestExportIndependentOrNot(t *testing.T) {
	s := InitialSetup(t)
	cliPath := s.cliPath
	generator := s.generator
	generator.tier = "M0"
	testPrefix := "test-"
	generator.generateDBUser(testPrefix)
	generator.generateCluster()
	expectAlertConfigs := false
	dictionary := resources.AtlasNameToKubernetesName()
	credentialName := resources.NormalizeAtlasName(generator.projectName+credSuffixTest, dictionary)

	for _, tc := range []struct {
		title                string
		independentResources bool
		expected             []runtime.Object
	}{
		{
			title:                "Exported without independentResources uses Kubernetes references",
			independentResources: false,
			expected: []runtime.Object{
				defaultTestProject(generator.projectName, "", expectedLabels, expectAlertConfigs),
				defaultTestAtlasConnSecret(credentialName, ""),
				defaultTestUser(generator.dbUser, generator.projectName, ""),
				defaultM0TestCluster(generator.clusterName, generator.clusterRegion, generator.projectName, ""),
			},
		},
		{
			title:                "Exported with independentResources uses IDs were supported",
			independentResources: true,
			expected: []runtime.Object{
				defaultTestProject(generator.projectName, "", expectedLabels, expectAlertConfigs),
				defaultTestAtlasConnSecret(credentialName, ""),
				defaultTestUserWithID(generator.dbUser, generator.projectName, generator.projectID, "", credentialName),
				defaultM0TestClusterWithID(
					generator.clusterName, generator.clusterRegion, generator.projectName, generator.projectID, "",
					credentialName,
				),
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			cmdArgs := []string{
				"kubernetes",
				"config",
				"generate",
				"--projectId",
				generator.projectID,
			}
			if tc.independentResources {
				cmdArgs = append(cmdArgs, "--independentResources")
			}
			cmd := exec.Command(cliPath, cmdArgs...)
			cmd.Env = os.Environ()
			resp, err := cmd.CombinedOutput()
			t.Log(string(resp))
			require.NoError(t, err, string(resp))
			var objects []runtime.Object
			objects, err = getK8SEntities(resp)
			// We want to filter spurious federated auth resources from other tests
			// as these are global resources across all projects.
			t.Log("pre filtered", len(objects))
			objects = filtered(objects).byKind(globalKinds...)
			t.Log("post filtered", len(objects))
			require.NoError(t, err, "should not fail on decode but got:\n"+string(resp))
			require.NotEmpty(t, objects)
			require.Equal(t, tc.expected, objects)
		})
	}
}

func TestExportPrivateEndpoint(t *testing.T) {
	s := InitialSetup(t)
	s.generator.generatePrivateEndpoint(awsEntity, "eu-central-1")

	credentialName := resources.NormalizeAtlasName(s.generator.projectName+credSuffixTest, resources.AtlasNameToKubernetesName())

	tests := map[string]struct {
		independentResources bool
		version              string
		expected             []runtime.Object
	}{
		"should export separate resource with internal reference for version with support": {
			independentResources: false,
			version:              features.LatestOperatorMajorVersion,
			expected: []runtime.Object{
				defaultTestProject(s.generator.projectName, "", expectedLabels, false),
				defaultTestAtlasConnSecret(credentialName, ""),
				defaultPrivateEndpoint(s.generator, false),
				referenceContainer(s.generator, "AWS", "EU_CENTRAL_1", "", expectedLabels, false),
			},
		},
		"should export separate resource with external reference for version with support": {
			independentResources: true,
			version:              features.LatestOperatorMajorVersion,
			expected: []runtime.Object{
				defaultTestProject(s.generator.projectName, "", expectedLabels, false),
				defaultTestAtlasConnSecret(credentialName, ""),
				defaultPrivateEndpoint(s.generator, true),
				referenceContainer(s.generator, "AWS", "EU_CENTRAL_1", "", expectedLabels, true),
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			operatorVersion := tc.version
			cmdArgs := []string{
				"kubernetes",
				"config",
				"generate",
				"--projectId",
				s.generator.projectID,
				"--operatorVersion",
				operatorVersion,
			}
			if tc.independentResources {
				cmdArgs = append(cmdArgs, "--independentResources")
			}
			cmd := exec.Command(s.cliPath, cmdArgs...) //nolint:gosec
			cmd.Env = os.Environ()
			resp, err := cmd.CombinedOutput()
			require.NoError(t, err, string(resp))

			var objects []runtime.Object
			objects, err = getK8SEntities(resp)
			objects = filtered(objects).byKind(globalKinds...)
			require.NoError(t, err, "should not fail on decode but got:\n"+string(resp))
			require.NotEmpty(t, objects)
			require.Equal(t, tc.expected, objects)
		})
	}
}

func TestExportIPAccessList(t *testing.T) {
	s := InitialSetup(t)
	expectedSubresource := []akov2project.IPAccessList{
		{
			CIDRBlock: "10.1.1.0/24",
		},
		{
			CIDRBlock: "172.16.0.0/16",
		},
		{
			CIDRBlock: "10.0.0.0/8",
		},
		{
			IPAddress: "198.51.100.42",
		},
		{
			IPAddress: "203.0.113.10",
		},
		{
			IPAddress: "192.168.100.233",
		},
	}
	credentialName := resources.NormalizeAtlasName(s.generator.projectName+credSuffixTest, resources.AtlasNameToKubernetesName())

	// Create access list entries for each IP and CIDR
	for _, entry := range expectedSubresource {
		var cmd *exec.Cmd

		if entry.IPAddress != "" {
			// #nosec G204
			cmd = exec.Command(s.atlasCliPath,
				accessListEntity,
				"create",
				entry.IPAddress,
				"--projectId",
				s.generator.projectID,
				"--type",
				"ipAddress",
				"-o=json")
		} else if entry.CIDRBlock != "" {
			// #nosec G204
			cmd = exec.Command(s.atlasCliPath,
				accessListEntity,
				"create",
				entry.CIDRBlock,
				"--projectId",
				s.generator.projectID,
				"--type",
				"cidrBlock",
				"-o=json")
		}

		if cmd != nil {
			cmd.Env = os.Environ()
			accessListResp, err := test.RunAndGetStdOut(cmd)
			require.NoError(t, err, string(accessListResp))
		}
	}

	tests := map[string]struct {
		independentResources bool
		version              string
		expected             []runtime.Object
	}{
		"should export separate resource with internal reference for version with support": {
			independentResources: false,
			version:              features.LatestOperatorMajorVersion,
			expected: []runtime.Object{
				defaultTestProject(s.generator.projectName, "", expectedLabels, false),
				defaultTestAtlasConnSecret(credentialName, ""),
				defaultIPAccessList(s.generator, false),
			},
		},
		"should export separate resource with external reference for version with support": {
			independentResources: true,
			version:              features.LatestOperatorMajorVersion,
			expected: []runtime.Object{
				defaultTestProject(s.generator.projectName, "", expectedLabels, false),
				defaultTestAtlasConnSecret(credentialName, ""),
				defaultIPAccessList(s.generator, true),
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			operatorVersion := tc.version
			cmdArgs := []string{
				"kubernetes",
				"config",
				"generate",
				"--projectId",
				s.generator.projectID,
				"--operatorVersion",
				operatorVersion,
			}
			if tc.independentResources {
				cmdArgs = append(cmdArgs, "--independentResources")
			}
			cmd := exec.Command(s.cliPath, cmdArgs...) //nolint:gosec
			cmd.Env = os.Environ()
			resp, err := cmd.CombinedOutput()
			require.NoError(t, err, string(resp))

			var objects []runtime.Object
			objects, err = getK8SEntities(resp)
			objects = filtered(objects).byKind(globalKinds...)
			require.NoError(t, err, "should not fail on decode but got:\n"+string(resp))
			require.NotEmpty(t, objects)
			require.Equal(t, tc.expected, objects)
		})
	}
}

func TestExportIntegrations(t *testing.T) {
	s := InitialSetup(t)
	operatorVersion := features.LatestOperatorMajorVersion
	datadogKey := "00000000000000000000000000000012"

	cmd := exec.Command(s.atlasCliPath,
		integrationsEntity,
		"create",
		datadogEntity,
		"--apiKey",
		datadogKey,
		"--projectId",
		s.generator.projectID,
		"-o=json")
	out, err := test.RunAndGetStdOut(cmd)
	require.NoError(t, err)
	reply := struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}{}
	require.NoError(t, json.Unmarshal(out, &reply))
	integrationID := reply.Results[0].ID

	datadogKeyMasked := "****************************0012"
	integrationName := strings.ToLower(s.generator.projectName) + "-datadog-integration"
	secretName := integrationName + "-secret"
	expectedProjectName := strings.ToLower(s.generator.projectName)

	expectedSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
			Labels: map[string]string{
				"atlas.mongodb.com/project-id":   s.generator.projectID,
				"atlas.mongodb.com/project-name": expectedProjectName,
				"atlas.mongodb.com/type":         "credentials",
			},
		},
		Data: map[string][]byte{
			"apiKey": ([]byte)(datadogKeyMasked),
		},
	}

	for _, tc := range []struct {
		title                string
		independentResources bool
		want                 []runtime.Object
	}{
		{
			title:                "independent integration",
			independentResources: true,
			want: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: integrationName,
						Labels: map[string]string{
							"mongodb.com/atlas-resource-version": features.LatestOperatorMajorVersion,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: s.generator.projectID,
							},
							ConnectionSecret: &akoapi.LocalObjectReference{
								Name: expectedProjectName + "-credentials",
							},
						},
						Type: "DATADOG",
						Datadog: &akov2.DatadogIntegration{
							APIKeySecretRef: akoapi.LocalObjectReference{
								Name: secretName,
							},
							Region:                       "US",
							SendCollectionLatencyMetrics: pointer.Get("disabled"),
							SendDatabaseMetrics:          pointer.Get("disabled"),
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						ID: integrationID,
					},
				},
				expectedSecret,
			},
		},
		{
			title:                "dependent integration",
			independentResources: false,
			want: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: integrationName,
						Labels: map[string]string{
							"mongodb.com/atlas-resource-version": features.LatestOperatorMajorVersion,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &akov2common.ResourceRefNamespaced{
								Name: expectedProjectName,
							},
						},
						Type: "DATADOG",
						Datadog: &akov2.DatadogIntegration{
							APIKeySecretRef: akoapi.LocalObjectReference{
								Name: secretName,
							},
							Region:                       "US",
							SendCollectionLatencyMetrics: pointer.Get("disabled"),
							SendDatabaseMetrics:          pointer.Get("disabled"),
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						ID: integrationID,
					},
				},
				expectedSecret,
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			cmdArgs := []string{
				"kubernetes",
				"config",
				"generate",
				"--projectId",
				s.generator.projectID,
				"--operatorVersion",
				operatorVersion,
			}
			if tc.independentResources {
				cmdArgs = append(cmdArgs, "--independentResources")
			}
			cmd := exec.Command(s.cliPath, cmdArgs...) //nolint:gosec
			cmd.Env = os.Environ()
			resp, err := cmd.CombinedOutput()
			require.NoError(t, err, string(resp))

			var objects []runtime.Object
			objects, err = getK8SEntities(resp)
			objects = filtered(objects).byKind(globalKinds...)
			require.NoError(t, err, "should not fail on decode but got:\n"+string(resp))
			require.NotEmpty(t, objects)
			credentialsName := resources.NormalizeAtlasName(
				strings.ToLower(s.generator.projectName)+"-credentials",
				resources.AtlasNameToKubernetesName(),
			)
			want := []runtime.Object{
				defaultTestProject(s.generator.projectName, "", expectedLabels, false),
				defaultTestAtlasConnSecret(credentialsName, ""),
			}
			want = append(want, tc.want...)
			require.Equal(t, want, objects)
		})
	}
}

func TestExportNetworkContainerAndPeerings(t *testing.T) {
	s := InitialSetup(t)
	operatorVersion := features.LatestOperatorMajorVersion

	awsContainerCIDR := "10.0.0.0/21"
	awsContainerID := s.generator.generateAWSContainer(awsContainerCIDR, "EU_CENTRAL_1")

	azureContainerCIDR := "10.128.0.0/21"
	azureContainerID := s.generator.generateAzureContainer(azureContainerCIDR, "EUROPE_NORTH")

	gcpContainerCIDR := "10.64.0.0/18"
	gcpContainerID := s.generator.generateGCPContainer(gcpContainerCIDR)

	awsAppVPC := s.generator.generateAWSNetworkVPC("10.128.0.0/21", "eu-south-2")
	defer s.generator.deleteAWSNetworkVPC(awsAppVPC)
	awsPeeringID := s.generator.generateAWSPeering(awsContainerID, awsAppVPC)
	defer s.generator.deletePeering(awsPeeringID)

	azureAppVPC := s.generator.generateAzureVPC("10.64.0.0/21", "northeurope")
	defer s.generator.deleteAzureVPC(azureAppVPC)
	azurePeeringID := s.generator.generateAzurePeering(azureContainerID, azureAppVPC)
	defer s.generator.deletePeering(azurePeeringID)

	gcpAppNetwork := s.generator.generateGCPNetworkVPC()
	defer s.generator.deleteGCPNetworkVPC(gcpAppNetwork)
	gcpPeeringID := s.generator.generateGCPPeering(gcpContainerID, gcpAppNetwork)
	defer s.generator.deletePeering(gcpPeeringID)

	for _, tc := range []struct {
		title                string
		independentResources bool
		want                 []runtime.Object
	}{
		{
			title:                "independent resource container",
			independentResources: true,
			want: []runtime.Object{
				defaultAWSContainer(s.generator, awsContainerID, awsContainerCIDR, true),
				defaultAzureContainer(s.generator, azureContainerID, azureContainerCIDR, true),
				defaultGCPContainer(s.generator, gcpContainerID, gcpContainerCIDR, true),

				defaultAWSPeering(s.generator, awsPeeringID, awsContainerID, awsAppVPC, true),
				defaultAzurePeering(s.generator, azurePeeringID, azureContainerID, azureAppVPC, true),
				defaultGCPPeering(s.generator, gcpPeeringID, gcpContainerID, gcpAppNetwork, true),
			},
		},
		{
			title:                "dependent resource container",
			independentResources: false,
			want: []runtime.Object{
				defaultAWSContainer(s.generator, awsContainerID, awsContainerCIDR, false),
				defaultAzureContainer(s.generator, azureContainerID, azureContainerCIDR, false),
				defaultGCPContainer(s.generator, gcpContainerID, gcpContainerCIDR, false),

				defaultAWSPeering(s.generator, awsPeeringID, awsContainerID, awsAppVPC, false),
				defaultAzurePeering(s.generator, azurePeeringID, azureContainerID, azureAppVPC, false),
				defaultGCPPeering(s.generator, gcpPeeringID, gcpContainerID, gcpAppNetwork, false),
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			cmdArgs := []string{
				"kubernetes",
				"config",
				"generate",
				"--projectId",
				s.generator.projectID,
				"--operatorVersion",
				operatorVersion,
			}
			if tc.independentResources {
				cmdArgs = append(cmdArgs, "--independentResources")
			}
			cmd := exec.Command(s.cliPath, cmdArgs...) //nolint:gosec
			cmd.Env = os.Environ()
			resp, err := cmd.CombinedOutput()
			require.NoError(t, err, string(resp))

			var objects []runtime.Object
			objects, err = getK8SEntities(resp)
			objects = filtered(objects).byKind(globalKinds...)
			require.NoError(t, err, "should not fail on decode but got:\n"+string(resp))
			require.NotEmpty(t, objects)
			credentialsName := resources.NormalizeAtlasName(
				strings.ToLower(s.generator.projectName)+"-credentials",
				resources.AtlasNameToKubernetesName(),
			)
			want := []runtime.Object{
				defaultTestProject(s.generator.projectName, "", expectedLabels, false),
				defaultTestAtlasConnSecret(credentialsName, ""),
			}
			want = append(want, tc.want...)
			require.Equal(t, want, objects)
		})
	}
}

func defaultAWSContainer(generator *atlasE2ETestGenerator, id, cidr string, independent bool) *akov2.AtlasNetworkContainer {
	resourceName := resources.NormalizeAtlasName(
		strings.ToLower(fmt.Sprintf("%s-container-aws-eucentral1", generator.projectName)),
		resources.AtlasNameToKubernetesName(),
	)
	return customContainer(generator, independent, resourceName, &akov2.AtlasNetworkContainerSpec{
		Provider: "AWS",
		AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
			ID:        id,
			Region:    "EU_CENTRAL_1",
			CIDRBlock: cidr,
		},
	})
}

func defaultAzureContainer(generator *atlasE2ETestGenerator, id, cidr string, independent bool) *akov2.AtlasNetworkContainer {
	resourceName := resources.NormalizeAtlasName(
		strings.ToLower(fmt.Sprintf("%s-container-azure-europenorth", generator.projectName)),
		resources.AtlasNameToKubernetesName(),
	)
	return customContainer(generator, independent, resourceName, &akov2.AtlasNetworkContainerSpec{
		Provider: "AZURE",
		AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
			ID:        id,
			Region:    "EUROPE_NORTH",
			CIDRBlock: cidr,
		},
	})
}

func defaultGCPContainer(generator *atlasE2ETestGenerator, id, cidr string, independent bool) *akov2.AtlasNetworkContainer {
	resourceName := resources.NormalizeAtlasName(
		strings.ToLower(fmt.Sprintf("%s-container-gcp-global", generator.projectName)),
		resources.AtlasNameToKubernetesName(),
	)
	return customContainer(generator, independent, resourceName, &akov2.AtlasNetworkContainerSpec{
		Provider: "GCP",
		AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
			ID:        id,
			CIDRBlock: cidr,
		},
	})
}

func customContainer(generator *atlasE2ETestGenerator, independent bool, resourceName string, spec *akov2.AtlasNetworkContainerSpec) *akov2.AtlasNetworkContainer {
	container := akov2.AtlasNetworkContainer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasNetworkContainer",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resourceName,
			Labels: expectedLabels,
		},
		Spec: *spec,
		Status: akov2status.AtlasNetworkContainerStatus{
			Common: api.Common{
				Conditions: []api.Condition{},
			},
		},
	}
	if independent {
		credentialsName := resources.NormalizeAtlasName(
			strings.ToLower(generator.projectName)+"-credentials",
			resources.AtlasNameToKubernetesName(),
		)
		container.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ExternalProjectRef: &akov2.ExternalProjectReference{
				ID: generator.projectID,
			},
			ConnectionSecret: &akoapi.LocalObjectReference{
				Name: credentialsName,
			},
		}
	} else {
		container.Spec.ProjectRef = &akov2common.ResourceRefNamespaced{
			Name: strings.ToLower(generator.projectName),
		}
	}
	return &container
}

func defaultAWSPeering(generator *atlasE2ETestGenerator, id, containerID string, vpc *awsVPC, independent bool) *akov2.AtlasNetworkPeering {
	resourceName := resources.NormalizeAtlasName(
		strings.ToLower(fmt.Sprintf("%s-peering-aws-%s-%s", generator.projectName, vpc.region, vpc.id)),
		resources.AtlasNameToKubernetesName(),
	)
	return customPeering(generator, independent, resourceName, &akov2.AtlasNetworkPeeringSpec{
		ContainerRef: akov2.ContainerDualReference{
			ID: containerID,
		},
		AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
			ID:       id,
			Provider: "AWS",
			AWSConfiguration: &akov2.AWSNetworkPeeringConfiguration{
				AccepterRegionName:  vpc.region,
				AWSAccountID:        os.Getenv("AWS_ACCOUNT_ID"),
				RouteTableCIDRBlock: vpc.cidr,
				VpcID:               vpc.id,
			},
		},
	})
}

func defaultAzurePeering(generator *atlasE2ETestGenerator, id, containerID string, vnet *azureVNet, independent bool) *akov2.AtlasNetworkPeering {
	subscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceName := resources.NormalizeAtlasName(
		strings.ToLower(fmt.Sprintf("%s-peering-azure-%s-%s", generator.projectName, subscriptionId, vnet.name)),
		resources.AtlasNameToKubernetesName(),
	)
	return customPeering(generator, independent, resourceName, &akov2.AtlasNetworkPeeringSpec{
		ContainerRef: akov2.ContainerDualReference{
			ID: containerID,
		},
		AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
			ID:       id,
			Provider: "AZURE",
			AzureConfiguration: &akov2.AzureNetworkPeeringConfiguration{
				AzureDirectoryID:    os.Getenv("AZURE_TENANT_ID"),
				AzureSubscriptionID: subscriptionId,
				ResourceGroupName:   os.Getenv("AZURE_RESOURCE_GROUP_NAME"),
				VNetName:            vnet.name,
			},
		},
	})
}

func defaultGCPPeering(generator *atlasE2ETestGenerator, id, containerID string, networkName string, independent bool) *akov2.AtlasNetworkPeering {
	gcpProject := os.Getenv("GOOGLE_PROJECT_ID")
	resourceName := resources.NormalizeAtlasName(
		strings.ToLower(fmt.Sprintf("%s-peering-gcp-%s-%s", generator.projectName, gcpProject, networkName)),
		resources.AtlasNameToKubernetesName(),
	)
	return customPeering(generator, independent, resourceName, &akov2.AtlasNetworkPeeringSpec{
		ContainerRef: akov2.ContainerDualReference{
			ID: containerID,
		},
		AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
			ID:       id,
			Provider: "GCP",
			GCPConfiguration: &akov2.GCPNetworkPeeringConfiguration{
				GCPProjectID: gcpProject,
				NetworkName:  networkName,
			},
		},
	})
}

func customPeering(generator *atlasE2ETestGenerator, independent bool, resourceName string, spec *akov2.AtlasNetworkPeeringSpec) *akov2.AtlasNetworkPeering {
	peering := akov2.AtlasNetworkPeering{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasNetworkPeering",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resourceName,
			Labels: expectedLabels,
		},
		Spec: *spec,
		Status: akov2status.AtlasNetworkPeeringStatus{
			Common: api.Common{
				Conditions: []api.Condition{},
			},
		},
	}
	if independent {
		credentialsName := resources.NormalizeAtlasName(
			strings.ToLower(generator.projectName)+"-credentials",
			resources.AtlasNameToKubernetesName(),
		)
		peering.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ExternalProjectRef: &akov2.ExternalProjectReference{
				ID: generator.projectID,
			},
			ConnectionSecret: &akoapi.LocalObjectReference{
				Name: credentialsName,
			},
		}
	} else {
		peering.Spec.ProjectRef = &akov2common.ResourceRefNamespaced{
			Name: strings.ToLower(generator.projectName),
		}
	}
	return &peering
}

func defaultPrivateEndpoint(generator *atlasE2ETestGenerator, independent bool) *akov2.AtlasPrivateEndpoint {
	pe := &akov2.AtlasPrivateEndpoint{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasPrivateEndpoint",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resources.NormalizeAtlasName(strings.ToLower(generator.projectName)+"-pe-aws-eucentral1", resources.AtlasNameToKubernetesName()),
			Labels: expectedLabels,
		},
		Spec: akov2.AtlasPrivateEndpointSpec{
			Provider: "AWS",
			Region:   "EU_CENTRAL_1",
		},
		Status: akov2status.AtlasPrivateEndpointStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}

	if independent {
		pe.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ExternalProjectRef: &akov2.ExternalProjectReference{
				ID: generator.projectID,
			},
			ConnectionSecret: &akoapi.LocalObjectReference{
				Name: resources.NormalizeAtlasName(strings.ToLower(generator.projectName)+"-credentials", resources.AtlasNameToKubernetesName()),
			},
		}
	} else {
		pe.Spec.ProjectRef = &akov2common.ResourceRefNamespaced{
			Name: strings.ToLower(generator.projectName),
		}
	}

	return pe
}

func defaultIPAccessList(generator *atlasE2ETestGenerator, independent bool) *akov2.AtlasIPAccessList {
	ial := &akov2.AtlasIPAccessList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasIPAccessList",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   resources.NormalizeAtlasName(strings.ToLower(generator.projectName)+"-ip-access-list", resources.AtlasNameToKubernetesName()),
			Labels: expectedLabels,
		},
		Spec: akov2.AtlasIPAccessListSpec{
			Entries: []akov2.IPAccessEntry{
				{
					CIDRBlock: "10.1.1.0/24",
				},
				{
					CIDRBlock: "172.16.0.0/16",
				},
				{
					CIDRBlock: "10.0.0.0/8",
				},
				{
					IPAddress: "198.51.100.42",
				},
				{
					IPAddress: "203.0.113.10",
				},
				{
					IPAddress: "192.168.100.233",
				},
			},
		},
		Status: akov2status.AtlasIPAccessListStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}

	if independent {
		ial.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ExternalProjectRef: &akov2.ExternalProjectReference{
				ID: generator.projectID,
			},
			ConnectionSecret: &akoapi.LocalObjectReference{
				Name: resources.NormalizeAtlasName(strings.ToLower(generator.projectName)+"-credentials", resources.AtlasNameToKubernetesName()),
			},
		}
	} else {
		ial.Spec.ProjectRef = &akov2common.ResourceRefNamespaced{
			Name: strings.ToLower(generator.projectName),
		}
	}

	return ial
}

type filtered []runtime.Object

func (f filtered) byKind(kinds ...string) []runtime.Object {
	result := f[:0]
	for _, obj := range f {
		for _, kind := range kinds {
			if obj.GetObjectKind().GroupVersionKind().Kind != kind {
				result = append(result, obj)
			}
		}
	}
	return result
}

func defaultTestProject(name, namespace string, labels map[string]string, alertConfigs bool) *akov2.AtlasProject {
	project := &akov2.AtlasProject{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasProject",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(name),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: akov2.AtlasProjectSpec{
			Name:             name,
			ConnectionSecret: &akov2common.ResourceRefNamespaced{},
			EncryptionAtRest: &akov2.EncryptionAtRest{
				AwsKms:         akov2.AwsKms{Enabled: pointer.Get(false), Valid: pointer.Get(false)},
				AzureKeyVault:  akov2.AzureKeyVault{Enabled: pointer.Get(false)},
				GoogleCloudKms: akov2.GoogleCloudKms{Enabled: pointer.Get(false)},
			},
			Auditing: &akov2.Auditing{},
			Settings: &akov2.ProjectSettings{
				IsCollectDatabaseSpecificsStatisticsEnabled: pointer.Get(true),
				IsDataExplorerEnabled:                       pointer.Get(true),
				IsPerformanceAdvisorEnabled:                 pointer.Get(true),
				IsRealtimePerformancePanelEnabled:           pointer.Get(true),
				IsSchemaAdvisorEnabled:                      pointer.Get(true),
			},
		},
		Status: akov2status.AtlasProjectStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}
	if alertConfigs {
		project.Spec.AlertConfigurations = []akov2.AlertConfiguration{
			defaultAlertConfig(),
		}
	}

	return project
}

func defaultAlertConfig() akov2.AlertConfiguration {
	return akov2.AlertConfiguration{
		Enabled:       true,
		EventTypeName: "NDS_X509_USER_AUTHENTICATION_MANAGED_USER_CERTS_EXPIRATION_CHECK",
		Threshold: &akov2.Threshold{
			Operator:  "LESS_THAN",
			Units:     "DAYS",
			Threshold: "30",
		},
		Notifications: []akov2.Notification{
			{
				APITokenRef:         akov2common.ResourceRefNamespaced{},
				DatadogAPIKeyRef:    akov2common.ResourceRefNamespaced{},
				DelayMin:            pointer.Get(0),
				EmailEnabled:        pointer.Get(true),
				FlowdockAPITokenRef: akov2common.ResourceRefNamespaced{},
				IntervalMin:         1440,
				SMSEnabled:          pointer.Get(false),
				OpsGenieAPIKeyRef:   akov2common.ResourceRefNamespaced{},
				ServiceKeyRef:       akov2common.ResourceRefNamespaced{},
				TypeName:            "GROUP",
				VictorOpsSecretRef:  akov2common.ResourceRefNamespaced{},
				Roles:               []string{"GROUP_OWNER"},
			},
		},
		MetricThreshold: &akov2.MetricThreshold{},
	}
}

func defaultTestAtlasConnSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(name),
			Namespace: namespace,
			Labels:    map[string]string{"atlas.mongodb.com/type": "credentials"},
		},
		Data: map[string][]byte{
			"orgId":         []uint8{},
			"privateApiKey": []uint8{},
			"publicApiKey":  []uint8{},
		},
		Type: "",
	}
}

func defaultTestUser(name, projectName, namespace string) *akov2.AtlasDatabaseUser {
	dictionary := resources.AtlasNameToKubernetesName()
	userName := resources.NormalizeAtlasName(strings.ToLower(fmt.Sprintf("%s-%s", projectName, name)), dictionary)
	return &akov2.AtlasDatabaseUser{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasDatabaseUser",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: namespace,
			Labels: map[string]string{
				"mongodb.com/atlas-resource-version": features.LatestOperatorMajorVersion,
			},
		},
		Spec: akov2.AtlasDatabaseUserSpec{
			ProjectDualReference: akov2.ProjectDualReference{
				ProjectRef: &akov2common.ResourceRefNamespaced{
					Name:      strings.ToLower(projectName),
					Namespace: namespace,
				},
			},
			DatabaseName: "$external",
			Roles: []akov2.RoleSpec{
				{
					RoleName:     "readAnyDatabase",
					DatabaseName: "admin",
				},
			},
			Username: name,
			X509Type: "MANAGED",
		},
		Status: akov2status.AtlasDatabaseUserStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}
}

func defaultTestUserWithID(name, projectName, projectID, namespace string, creds string) *akov2.AtlasDatabaseUser {
	user := defaultTestUser(name, projectName, namespace)
	user.Spec.ProjectDualReference.ProjectRef = nil
	user.Spec.ProjectDualReference.ExternalProjectRef = &akov2.ExternalProjectReference{
		ID: projectID,
	}
	user.Spec.ProjectDualReference.ConnectionSecret = &akoapi.LocalObjectReference{
		Name: creds,
	}
	return user
}

func defaultM0TestCluster(name, region, projectName, namespace string) *akov2.AtlasDeployment {
	dictionary := resources.AtlasNameToKubernetesName()
	clusterName := resources.NormalizeAtlasName(strings.ToLower(fmt.Sprintf("%s-%s", projectName, name)), dictionary)
	return &akov2.AtlasDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasDeployment",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			Labels: map[string]string{
				"mongodb.com/atlas-resource-version": features.LatestOperatorMajorVersion,
			},
		},
		Spec: akov2.AtlasDeploymentSpec{
			ProjectDualReference: akov2.ProjectDualReference{
				ProjectRef: &akov2common.ResourceRefNamespaced{
					Name:      strings.ToLower(projectName),
					Namespace: namespace,
				},
			},
			DeploymentSpec: &akov2.AdvancedDeploymentSpec{
				ClusterType: "REPLICASET",
				Name:        name,
				Paused:      pointer.Get(false),
				ReplicationSpecs: []*akov2.AdvancedReplicationSpec{
					{
						NumShards: 1,
						ZoneName:  "Zone 1",
						RegionConfigs: []*akov2.AdvancedRegionConfig{
							{
								ElectableSpecs: &akov2.Specs{
									InstanceSize: "M0",
								},
								BackingProviderName: "AWS",
								Priority:            pointer.Get(7),
								ProviderName:        "TENANT",
								RegionName:          region,
							},
						},
					},
				},
				RootCertType:         "ISRGROOTX1",
				VersionReleaseSystem: "LTS",
			},
			ProcessArgs: &akov2.ProcessArgs{
				MinimumEnabledTLSProtocol: "TLS1_2",
				JavascriptEnabled:         pointer.Get(true),
				NoTableScan:               pointer.Get(false),
			},
		},
		Status: akov2status.AtlasDeploymentStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}
}

func defaultM0TestClusterWithID(name, region, projectName, projectID, namespace, creds string) *akov2.AtlasDeployment {
	deployment := defaultM0TestCluster(name, region, projectName, namespace)
	deployment.Spec.ProjectDualReference.ProjectRef = nil
	deployment.Spec.ProjectDualReference.ExternalProjectRef = &akov2.ExternalProjectReference{
		ID: projectID,
	}
	deployment.Spec.ProjectDualReference.ConnectionSecret = &akoapi.LocalObjectReference{
		Name: creds,
	}
	return deployment
}

func TestFederatedAuthTest(t *testing.T) {
	t.Run("PreRequisite Get the federation setting ID", func(t *testing.T) {
		s := InitialSetup(t)
		atlasCliPath := s.atlasCliPath
		cmd := exec.Command(atlasCliPath,
			federatedAuthenticationEntity,
			federationSettingsEntity,
			"describe",
			"-o=json",
		)

		cmd.Env = os.Environ()
		resp, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err, string(resp))

		var settings atlasv2.OrgFederationSettings
		require.NoError(t, json.Unmarshal(resp, &settings))

		a := assert.New(t)
		a.NotEmpty(settings)
		federationSettingsID = settings.GetId()
		a.NotEmpty(federationSettingsID, "no federation settings was present")
		identityProviderStatus = settings.GetIdentityProviderStatus()
	})
	t.Run("List SAML IdPs", func(_ *testing.T) {
		if identityProviderStatus != activeStatus {
			s := InitialSetup(t)
			atlasCliPath := s.atlasCliPath
			cmd := exec.Command(atlasCliPath,
				federatedAuthenticationEntity,
				federationSettingsEntity,
				identityProviderEntity,
				"list",
				"--federationSettingsId",
				federationSettingsID,
				"--protocol",
				"SAML",
				"-o=json",
			)

			cmd.Env = os.Environ()
			resp, err := test.RunAndGetStdOut(cmd)
			require.NoError(t, err, string(resp))

			var providers atlasv2.PaginatedFederationIdentityProvider
			require.NoError(t, json.Unmarshal(resp, &providers))
			a := assert.New(t)
			a.True(providers.HasResults())
			providersList := providers.GetResults()
			samlIdentityProviderID = providersList[0].GetOktaIdpId()
		}
	})
	t.Run("PreRequisite Connect SAML IdP", func(t *testing.T) {
		if identityProviderStatus != activeStatus && samlIdentityProviderID != "" {
			s := InitialSetup(t)
			atlasCliPath := s.atlasCliPath
			cmd := exec.Command(atlasCliPath,
				federatedAuthenticationEntity,
				federationSettingsEntity,
				connectedOrgsConfigsEntity,
				"connect",
				"--identityProviderId",
				samlIdentityProviderID,
				"--federationSettingsId",
				federationSettingsID,
				"--protocol",
				"SAML",
				"-o=json",
			)

			cmd.Env = os.Environ()
			resp, err := test.RunAndGetStdOut(cmd)
			require.NoError(t, err, string(resp))

			var config atlasv2.ConnectedOrgConfig
			require.NoError(t, json.Unmarshal(resp, &config))
			assert.NotNil(t, config.GetIdentityProviderId())
		}
	})
	t.Run("Prerequisite Check active SAML configuration", func(t *testing.T) {
		if identityProviderStatus != activeStatus {
			s := InitialSetup(t)
			atlasCliPath := s.atlasCliPath
			cmd := exec.Command(atlasCliPath,
				federatedAuthenticationEntity,
				federationSettingsEntity,
				"describe",
				"-o=json",
			)

			cmd.Env = os.Environ()
			resp, err := test.RunAndGetStdOut(cmd)
			require.NoError(t, err, string(resp))

			var settings atlasv2.OrgFederationSettings
			require.NoError(t, json.Unmarshal(resp, &settings))

			a := assert.New(t)
			a.NotEmpty(settings)
			federationSettingsID = settings.GetId()
			a.NotEmpty(federationSettingsID, "no federation settings was present")
			a.NotEmpty(settings.IdentityProviderId, "no SAML IdP was found")
			a.Equal(activeStatus, settings.GetIdentityProviderStatus(), "no active SAML IdP present for this federation")
			identityProviderStatus = settings.GetIdentityProviderStatus()
		}
	})
	t.Run("Config generate for federated auth", func(t *testing.T) {
		if identityProviderStatus != activeStatus {
			t.Fatalf("There is no need to check this test since there is no SAML IdP configured and active")
		}
		dictionary := resources.AtlasNameToKubernetesName()
		s := InitialSetup(t)
		cliPath := s.cliPath
		generator := s.generator
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()
		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))
		var objects []runtime.Object
		objects, err = getK8SEntities(resp)

		a := assert.New(t)
		expectedInCloudDev := &akov2.AtlasFederatedAuth{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AtlasFederatedAuth",
				APIVersion: "atlas.mongodb.com/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-%s", s.generator.projectName, federationSettingsID), dictionary),
				Namespace: targetNamespace,
			},
			Spec: akov2.AtlasFederatedAuthSpec{
				ConnectionSecretRef: akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(s.generator.projectName+credSuffixTest, dictionary),
					Namespace: targetNamespace,
				},
				Enabled:                  true,
				DomainAllowList:          []string{"iam-test-domain-dev.com"},
				PostAuthRoleGrants:       []string{"ORG_OWNER"},
				DomainRestrictionEnabled: pointer.Get(false),
				SSODebugEnabled:          pointer.Get(true),
				RoleMappings:             nil,
			},
			Status: akov2status.AtlasFederatedAuthStatus{
				Common: akoapi.Common{
					Conditions: []akoapi.Condition{},
				},
			},
		}
		fedAuths := federatedAuthentification(objects)
		require.Len(t, fedAuths, 1)
		expectedInCloudQA := &akov2.AtlasFederatedAuth{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AtlasFederatedAuth",
				APIVersion: "atlas.mongodb.com/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-%s", s.generator.projectName, federationSettingsID), dictionary),
				Namespace: targetNamespace,
			},
			Spec: akov2.AtlasFederatedAuthSpec{
				ConnectionSecretRef: akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(s.generator.projectName+credSuffixTest, dictionary),
					Namespace: targetNamespace,
				},
				Enabled: true,
				DomainAllowList: []string{
					"qa-27092023.com",
					"cloud-qa.mongodb.com",
					"mongodb.com",
				},
				PostAuthRoleGrants:       []string{"ORG_MEMBER"},
				DomainRestrictionEnabled: pointer.Get(true),
				SSODebugEnabled:          pointer.Get(false),
				RoleMappings: []akov2.RoleMapping{
					{
						ExternalGroupName: "test",
						RoleAssignments: []akov2.RoleAssignment{
							{
								ProjectName: "",
								Role:        "ORG_BILLING_ADMIN",
							},
							{
								ProjectName: "",
								Role:        "ORG_GROUP_CREATOR",
							},
							{
								ProjectName: "",
								Role:        "ORG_OWNER",
							},
						},
					},
				},
			},
			Status: akov2status.AtlasFederatedAuthStatus{
				Common: akoapi.Common{
					Conditions: []akoapi.Condition{},
				},
			},
		}
		expected := expectedInCloudDev
		if isQAEnv(os.Getenv("MCLI_OPS_MANAGER_URL")) {
			expected = expectedInCloudQA
		}
		a.Equal(expected, normalizedFedAuth(fedAuths[0]))
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)
		secret, found := findSecret(objects)
		require.True(t, found, "secret is not found in results")
		a.Equal(targetNamespace, secret.Namespace)
	})
}

func normalizedFedAuth(fedAuth *akov2.AtlasFederatedAuth) *akov2.AtlasFederatedAuth {
	for _, rm := range fedAuth.Spec.RoleMappings {
		slices.SortFunc(rm.RoleAssignments, func(a, b akov2.RoleAssignment) int {
			return strings.Compare(a.ProjectName+a.Role, b.ProjectName+b.Role)
		})
	}
	slices.SortFunc(fedAuth.Spec.RoleMappings, func(a, b akov2.RoleMapping) int {
		return strings.Compare(a.ExternalGroupName, b.ExternalGroupName)
	})
	return fedAuth
}

func federatedAuthentification(objects []runtime.Object) []*akov2.AtlasFederatedAuth {
	var ds []*akov2.AtlasFederatedAuth
	for i := range objects {
		d, ok := objects[i].(*akov2.AtlasFederatedAuth)
		if ok {
			ds = append(ds, d)
		}
	}
	return ds
}

func TestEmptyProject(t *testing.T) {
	s := InitialSetup(t)
	cliPath := s.cliPath
	generator := s.generator
	expectedProject := s.expectedProject

	t.Run("Generate valid resources of ONE project", func(t *testing.T) {
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--orgId", "", // Empty org id does not make it fail
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)

		checkProject(t, objects, expectedProject)
		secret, found := findSecret(objects)
		require.True(t, found, "Secret is not found in results")
		assert.Equal(t, targetNamespace, secret.Namespace)
	})
}

func TestProjectWithNonDefaultSettings(t *testing.T) {
	s := InitialSetup(t)
	cliPath := s.cliPath
	atlasCliPath := s.atlasCliPath
	generator := s.generator
	expectedProject := s.expectedProject
	expectedProject.Spec.Settings.IsCollectDatabaseSpecificsStatisticsEnabled = pointer.Get(false)

	t.Run("Change project settings and generate", func(t *testing.T) {
		cmd := exec.Command(atlasCliPath,
			projectsEntity,
			settingsEntity,
			"update",
			"--disableCollectDatabaseSpecificsStatistics",
			"-o=json",
			"--projectId",
			generator.projectID)
		cmd.Env = os.Environ()
		settingsResp, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err, string(settingsResp))

		cmd = exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err)
		require.NotEmpty(t, objects)
		checkProject(t, objects, expectedProject)
	})
}

func TestProjectWithNonDefaultAlertConf(t *testing.T) {
	dictionary := resources.AtlasNameToKubernetesName()
	s := InitialSetup(t)
	cliPath := s.cliPath
	atlasCliPath := s.atlasCliPath
	generator := s.generator
	expectedProject := s.expectedProject

	newAlertConfig := akov2.AlertConfiguration{
		Threshold:       &akov2.Threshold{},
		MetricThreshold: &akov2.MetricThreshold{},
		EventTypeName:   "HOST_DOWN",
		Enabled:         true,
		Notifications: []akov2.Notification{
			{
				TypeName:     group,
				IntervalMin:  intervalMin,
				DelayMin:     pointer.Get(delayMin),
				SMSEnabled:   pointer.Get(false),
				EmailEnabled: pointer.Get(true),
				APITokenRef: akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(expectedProject.Name+"-api-token-0", dictionary),
					Namespace: targetNamespace,
				},
				DatadogAPIKeyRef: akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(expectedProject.Name+"-datadog-api-key-0", dictionary),
					Namespace: targetNamespace,
				},
				OpsGenieAPIKeyRef: akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(expectedProject.Name+"-ops-genie-api-key-0", dictionary),
					Namespace: targetNamespace,
				},
				ServiceKeyRef: akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(expectedProject.Name+"-service-key-0", dictionary),
					Namespace: targetNamespace,
				},
				VictorOpsSecretRef: akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(expectedProject.Name+"-victor-ops-credentials-0", dictionary),
					Namespace: targetNamespace,
				},
			},
		},
		Matchers: []akov2.Matcher{
			{
				FieldName: "HOSTNAME",
				Operator:  "CONTAINS",
				Value:     "some-name",
			},
		},
	}
	expectedProject.Spec.AlertConfigurations = []akov2.AlertConfiguration{
		newAlertConfig,
	}

	t.Run("Change project alert config and generate", func(t *testing.T) {
		cmd := exec.Command(atlasCliPath,
			alertsEntity,
			configEntity,
			"create",
			"--projectId",
			generator.projectID,
			"--event",
			newAlertConfig.EventTypeName,
			fmt.Sprintf("--enabled=%t", newAlertConfig.Enabled),
			"--notificationType",
			newAlertConfig.Notifications[0].TypeName,
			"--notificationIntervalMin",
			strconv.Itoa(newAlertConfig.Notifications[0].IntervalMin),
			"--notificationDelayMin",
			strconv.Itoa(*newAlertConfig.Notifications[0].DelayMin),
			fmt.Sprintf("--notificationSmsEnabled=%v", pointer.GetOrZero(newAlertConfig.Notifications[0].SMSEnabled)),
			fmt.Sprintf("--notificationEmailEnabled=%v", pointer.GetOrZero(newAlertConfig.Notifications[0].EmailEnabled)),
			"--matcherFieldName",
			newAlertConfig.Matchers[0].FieldName,
			"--matcherOperator",
			newAlertConfig.Matchers[0].Operator,
			"--matcherValue",
			newAlertConfig.Matchers[0].Value,
			"-o=json")
		cmd.Env = os.Environ()
		alertConfigResp, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err, string(alertConfigResp))

		cmd = exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err)
		require.NotEmpty(t, objects)
		checkProject(t, objects, expectedProject)
	})
}

// func TestProjectWithOrgSettings(t *testing.T) {
// 	s := InitialSetup(t)
// 	cliPath := s.cliPath
// 	generator := s.generator
// 	expectedProject := s.expectedProject
//
// 	t.Run("Should export OrgSettings", func(t *testing.T) {
// 		cmd := exec.Command(cliPath,
// 			"kubernetes",
// 			"config",
// 			"generate",
// 			"--projectId",
// 			generator.projectID,
// 			"--targetNamespace",
// 			targetNamespace,
// 			"--includeSecrets")
// 		cmd.Env = os.Environ()
//
// 		resp, err := test.RunAndGetStdOut(cmd)
// 		t.Log(string(resp))
// 		require.NoError(t, err, string(resp))
//
// 		var objects []runtime.Object
// 		objects, err = getK8SEntities(resp)
// 		require.NoError(t, err)
// 		require.NotEmpty(t, objects)
// 		checkProject(t, objects, expectedProject)
// 		checkOrgSettings(t, objects)
// 	})
// }
//
// func checkOrgSettings(t *testing.T, output []runtime.Object) {
// 	t.Helper()
// 	found := false
// 	var orgSettings *akov2.AtlasOrgSettings
// 	for i := range output {
// 		p, ok := output[i].(*akov2.AtlasOrgSettings)
// 		if ok {
// 			found = true
// 			orgSettings = p
// 			break
// 		}
// 	}
// 	require.True(t, found, "AtlasOrgSettings is not found in results")
// 	secretName := orgSettings.Spec.ConnectionSecretRef.Name
// 	found = false
// 	for i := range output {
// 		p, ok := output[i].(*corev1.Secret)
// 		if ok && p.GetName() == secretName {
// 			found = true
// 			break
// 		}
// 	}
// 	require.True(t, found, "AtlasOrgSettings secret is not found in results")
// }

func TestProjectWithAccessRole(t *testing.T) {
	s := InitialSetup(t)
	cliPath := s.cliPath
	atlasCliPath := s.atlasCliPath
	generator := s.generator
	expectedProject := s.expectedProject

	newIPAccess := akov2.CloudProviderAccessRole{
		ProviderName: string(akov2provider.ProviderAWS),
	}
	expectedProject.Spec.CloudProviderAccessRoles = []akov2.CloudProviderAccessRole{
		newIPAccess,
	}

	t.Run("Add access role to the project", func(t *testing.T) {
		cmd := exec.Command(atlasCliPath,
			cloudProvidersEntity,
			accessRolesEntity,
			awsEntity,
			"create",
			"--projectId",
			generator.projectID,
			"-o=json")
		cmd.Env = os.Environ()
		accessRoleResp, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err, string(accessRoleResp))

		cmd = exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err)
		require.NotEmpty(t, objects)
		checkProject(t, objects, expectedProject)
	})
}

func TestProjectWithCustomRole(t *testing.T) {
	s := InitialSetup(t)
	cliPath := s.cliPath
	atlasCliPath := s.atlasCliPath
	generator := s.generator
	expectedProject := s.expectedProject

	newCustomRole := akov2.CustomRole{
		Name: "test-role",
		Actions: []akov2.Action{
			{
				Name: "FIND",
				Resources: []akov2.Resource{
					{
						Database:   pointer.Get("test-db"),
						Collection: pointer.Get(""),
						Cluster:    pointer.Get(false),
					},
				},
			},
		},
	}
	expectedProject.Spec.CustomRoles = []akov2.CustomRole{
		newCustomRole,
	}

	t.Run("Add custom role to the project", func(t *testing.T) {
		cmd := exec.Command(atlasCliPath,
			customDBRoleEntity,
			"create",
			newCustomRole.Name,
			"--privilege",
			fmt.Sprintf("%s@%s", newCustomRole.Actions[0].Name, *newCustomRole.Actions[0].Resources[0].Database),
			"--projectId",
			generator.projectID,
			"-o=json")
		cmd.Env = os.Environ()
		dbRoleResp, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err, string(dbRoleResp))

		cmd = exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)
		expectedProject.Spec.CustomRoles = nil
		verifyCustomRole(t, objects, &akov2.AtlasCustomRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AtlasCustomRole",
				APIVersion: "atlas.mongodb.com/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-custom-role-%s", expectedProject.Name, newCustomRole.Name), resources.AtlasNameToKubernetesName()),
				Namespace: expectedProject.Namespace,
				Labels: map[string]string{
					"mongodb.com/atlas-resource-version": features.LatestOperatorMajorVersion,
				},
			},
			Spec: akov2.AtlasCustomRoleSpec{
				ProjectDualReference: akov2.ProjectDualReference{
					ProjectRef: &akov2common.ResourceRefNamespaced{
						Name:      expectedProject.Name,
						Namespace: expectedProject.Namespace,
					},
				},
				Role: akov2.CustomRole{
					Name: "test-role",
					Actions: []akov2.Action{
						{
							Name: "FIND",
							Resources: []akov2.Resource{
								{
									Database:   pointer.Get("test-db"),
									Collection: pointer.Get(""),
									Cluster:    pointer.Get(false),
								},
							},
						},
					},
				},
			},
			Status: akov2status.AtlasCustomRoleStatus{
				Common: akoapi.Common{
					Conditions: []akoapi.Condition{},
				},
			},
		},
		)
		checkProject(t, objects, expectedProject)
	})
}

func verifyCustomRole(t *testing.T, objects []runtime.Object, expectedRole *akov2.AtlasCustomRole) {
	t.Helper()
	var role *akov2.AtlasCustomRole
	for i := range objects {
		d, ok := objects[i].(*akov2.AtlasCustomRole)
		if ok {
			role = d
			break
		}
	}
	assert.Equal(t, expectedRole, role)
}

// TestProjectWithIntegration tests integratiosn embedded in the project
// TODO: remove test when 2.8 is deprecated, last version with embedded integrations
func TestProjectWithIntegration(t *testing.T) {
	s := InitialSetup(t)
	cliPath := s.cliPath
	atlasCliPath := s.atlasCliPath
	generator := s.generator
	operatorVersion := "2.8.2"
	expectedProject := referenceProject(s.generator.projectName, targetNamespace, map[string]string{
		features.ResourceVersion: operatorVersion,
	})

	datadogKey := "00000000000000000000000000000012"
	newIntegration := akov2project.Integration{
		Type:   datadogEntity,
		Region: "US", // it's a default value
		APIKeyRef: akov2common.ResourceRefNamespaced{
			Namespace: targetNamespace,
			Name:      fmt.Sprintf("%s-integration-%s", generator.projectID, strings.ToLower(datadogEntity)),
		},
	}
	expectedProject.Spec.Integrations = []akov2project.Integration{
		newIntegration,
	}
	expectedProject.Labels[features.ResourceVersion] = operatorVersion

	t.Run("Add integration to the project", func(t *testing.T) {
		cmd := exec.Command(atlasCliPath,
			integrationsEntity,
			"create",
			datadogEntity,
			"--apiKey",
			datadogKey,
			"--projectId",
			generator.projectID,
			"-o=json")
		cmd.Env = os.Environ()
		_, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err)

		cmd = exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets",
			"--operatorVersion", operatorVersion)
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object

		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)
		objects = filtered(objects).byKind(globalKinds...)

		checkProject(t, objects, expectedProject)
		assert.Len(t, objects, 3, "should have 3 objects in the output: project, integration secret, atlas secret")
		integrationSecret := objects[1].(*corev1.Secret)
		password, ok := integrationSecret.Data["password"]
		assert.True(t, ok, "should have password field in the integration secret")
		assert.True(t, compareStingsWithHiddenPart(datadogKey, string(password), uint8('*')), "should have correct password in the integration secret")
	})
}

func TestProjectWithMaintenanceWindow(t *testing.T) {
	s := InitialSetup(t)
	cliPath := s.cliPath
	atlasCliPath := s.atlasCliPath
	generator := s.generator
	expectedProject := s.expectedProject
	newMaintenanceWindow := akov2project.MaintenanceWindow{
		DayOfWeek: 1,
		HourOfDay: 1,
	}
	expectedProject.Spec.MaintenanceWindow = newMaintenanceWindow
	expectedProject.Spec.AlertConfigurations = nil

	t.Run("Add integration to the project", func(t *testing.T) {
		cmd := exec.Command(atlasCliPath,
			maintenanceEntity,
			"update",
			"--dayOfWeek",
			strconv.Itoa(newMaintenanceWindow.DayOfWeek),
			"--hourOfDay",
			strconv.Itoa(newMaintenanceWindow.HourOfDay),
			"--projectId",
			generator.projectID,
			"-o=json")
		cmd.Env = os.Environ()
		_, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err)

		cmd = exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)
		checkProject(t, objects, expectedProject)
	})
}

func TestProjectWithPrivateEndpoint_Azure(t *testing.T) {
	s := InitialSetup(t)
	s.generator.generatePrivateEndpoint(azureEntity, "northeurope")

	credentialName := resources.NormalizeAtlasName(s.generator.projectName+credSuffixTest, resources.AtlasNameToKubernetesName())
	expected := []runtime.Object{
		defaultTestProject(s.generator.projectName, targetNamespace, expectedLabels, false),
		defaultTestAtlasConnSecret(credentialName, targetNamespace),
		&akov2.AtlasPrivateEndpoint{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AtlasPrivateEndpoint",
				APIVersion: "atlas.mongodb.com/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.NormalizeAtlasName(strings.ToLower(s.generator.projectName)+"-pe-azure-europenorth", resources.AtlasNameToKubernetesName()),
				Namespace: targetNamespace,
				Labels:    expectedLabels,
			},
			Spec: akov2.AtlasPrivateEndpointSpec{
				Provider: "AZURE",
				Region:   "EUROPE_NORTH",
				ProjectDualReference: akov2.ProjectDualReference{
					ProjectRef: &akov2common.ResourceRefNamespaced{
						Name:      strings.ToLower(s.generator.projectName),
						Namespace: targetNamespace,
					},
				},
			},
			Status: akov2status.AtlasPrivateEndpointStatus{
				Common: akoapi.Common{
					Conditions: []akoapi.Condition{},
				},
			},
		},
		referenceContainer(s.generator, "AZURE", "EUROPE_NORTH", targetNamespace, expectedLabels, false),
	}

	t.Run("Add network peer to the project", func(t *testing.T) {
		cmd := exec.Command(s.cliPath, //nolint:gosec
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			s.generator.projectID,
			"--targetNamespace",
			targetNamespace)
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		objects = filtered(objects).byKind(globalKinds...)
		require.NoError(t, err, "should not fail on decode but got:\n"+string(resp))
		require.NotEmpty(t, objects)
		require.Equal(t, expected, objects)
	})
}

func referenceContainer(g *atlasE2ETestGenerator, provider, region, namespace string, labels map[string]string, independent bool) *akov2.AtlasNetworkContainer {
	c := &akov2.AtlasNetworkContainer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasNetworkContainer",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.NormalizeAtlasName(
				fmt.Sprintf(""+
					"%s-container-%s-%s",
					g.projectName,
					provider,
					strings.ToLower(strings.ReplaceAll(region, "_", "")),
				),
				resources.AtlasNameToKubernetesName(),
			),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: akov2.AtlasNetworkContainerSpec{
			Provider: provider,
			AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
				ID:        g.containerID,
				Region:    region,
				CIDRBlock: "192.168.248.0/21",
			},
		},
		Status: akov2status.AtlasNetworkContainerStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}

	if independent {
		c.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ExternalProjectRef: &akov2.ExternalProjectReference{
				ID: g.projectID,
			},
			ConnectionSecret: &akoapi.LocalObjectReference{
				Name: resources.NormalizeAtlasName(strings.ToLower(g.projectName)+"-credentials", resources.AtlasNameToKubernetesName()),
			},
		}
	} else {
		c.Spec.ProjectRef = &akov2common.ResourceRefNamespaced{
			Name:      strings.ToLower(g.projectName),
			Namespace: namespace,
		}
	}

	return c
}

func TestProjectAndTeams(t *testing.T) {
	s := InitialSetupWithTeam(t)
	cliPath := s.cliPath
	atlasCliPath := s.atlasCliPath
	generator := s.generator
	expectedProject := s.expectedProject

	teamRole := "GROUP_OWNER"

	t.Run("Add team to project", func(t *testing.T) {
		expectedTeam := referenceTeam(generator.teamName, targetNamespace, []akov2.TeamUser{
			akov2.TeamUser(generator.teamUser),
		}, generator.projectName, expectedLabels)

		expectedProject.Spec.Teams = []akov2.Team{
			{
				TeamRef: akov2common.ResourceRefNamespaced{
					Namespace: targetNamespace,
					Name:      expectedTeam.Name,
				},
				Roles: []akov2.TeamRole{
					akov2.TeamRole(teamRole),
				},
			},
		}

		cmd := exec.Command(atlasCliPath,
			projectsEntity,
			teamsEntity,
			"add",
			generator.teamID,
			"--role",
			teamRole,
			"--projectId",
			generator.projectID,
			"-o=json")
		cmd.Env = os.Environ()
		resp, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err, string(resp))

		cmd = exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err = test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)
		checkProject(t, objects, expectedProject)
		t.Run("Team is created", func(t *testing.T) {
			for _, obj := range objects {
				if team, ok := obj.(*akov2.AtlasTeam); ok {
					assert.Equal(t, expectedTeam, team)
				}
			}
		})
	})
}

func TestProjectWithStreamsProcessing(t *testing.T) {
	s := InitialSetup(t)
	s.generator.generateStreamsInstance("test-instance")
	s.generator.generateStreamsConnection("test-connection")

	cliPath := s.cliPath
	generator := s.generator

	t.Run("should export streams instance and connection resources", func(t *testing.T) {
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			generator.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)

		for i := range objects {
			object := objects[i]

			if instance, ok := object.(*akov2.AtlasStreamInstance); ok {
				assert.Equal(
					t,
					&akov2.AtlasStreamInstance{
						TypeMeta: metav1.TypeMeta{
							Kind:       "AtlasStreamInstance",
							APIVersion: "atlas.mongodb.com/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: resources.NormalizeAtlasName(
								fmt.Sprintf("%s-%s", generator.projectName, generator.streamInstanceName),
								resources.AtlasNameToKubernetesName(),
							),
							Namespace: targetNamespace,
							Labels: map[string]string{
								features.ResourceVersion: features.LatestOperatorMajorVersion,
							},
						},
						Spec: akov2.AtlasStreamInstanceSpec{
							Name: generator.streamInstanceName,
							Config: akov2.Config{
								Provider: "AWS",
								Region:   "VIRGINIA_USA",
								Tier:     "SP30",
							},
							Project: akov2common.ResourceRefNamespaced{
								Name:      resources.NormalizeAtlasName(generator.projectName, resources.AtlasNameToKubernetesName()),
								Namespace: targetNamespace,
							},
							ConnectionRegistry: []akov2common.ResourceRefNamespaced{
								{
									Name: resources.NormalizeAtlasName(
										fmt.Sprintf("%s-%s-%s", generator.projectName, generator.streamInstanceName, generator.streamConnectionName),
										resources.AtlasNameToKubernetesName(),
									),
									Namespace: targetNamespace,
								},
							},
						},
						Status: akov2status.AtlasStreamInstanceStatus{
							Common: akoapi.Common{
								Conditions: []akoapi.Condition{},
							},
						},
					},
					instance,
				)
			}

			if connection, ok := object.(*akov2.AtlasStreamConnection); ok {
				assert.Equal(
					t,
					&akov2.AtlasStreamConnection{
						TypeMeta: metav1.TypeMeta{
							Kind:       "AtlasStreamConnection",
							APIVersion: "atlas.mongodb.com/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: resources.NormalizeAtlasName(
								fmt.Sprintf("%s-%s-%s", generator.projectName, generator.streamInstanceName, generator.streamConnectionName),
								resources.AtlasNameToKubernetesName(),
							),
							Namespace: targetNamespace,
							Labels: map[string]string{
								features.ResourceVersion: features.LatestOperatorMajorVersion,
							},
						},
						Spec: akov2.AtlasStreamConnectionSpec{
							Name:           generator.streamConnectionName,
							ConnectionType: "Kafka",
							KafkaConfig: &akov2.StreamsKafkaConnection{
								Authentication: akov2.StreamsKafkaAuthentication{
									Mechanism: "SCRAM-256",
									Credentials: akov2common.ResourceRefNamespaced{
										Name: resources.NormalizeAtlasName(
											fmt.Sprintf("%s-%s-%s-userpass", generator.projectName, generator.streamInstanceName, generator.streamConnectionName),
											resources.AtlasNameToKubernetesName(),
										),
										Namespace: targetNamespace,
									},
								},
								BootstrapServers: "example.com:8080,fraud.example.com:8000",
								Security: akov2.StreamsKafkaSecurity{
									Protocol: "PLAINTEXT",
								},
								Config: map[string]string{"auto.offset.reset": "earliest"},
							},
						},
						Status: akov2status.AtlasStreamConnectionStatus{
							Common: akoapi.Common{
								Conditions: []akoapi.Condition{},
							},
						},
					},
					connection,
				)
			}

			if secret, ok := object.(*corev1.Secret); ok && strings.Contains(secret.Name, "userpass") {
				assert.Equal(
					t,
					&corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Secret",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: resources.NormalizeAtlasName(
								fmt.Sprintf("%s-%s-%s-userpass", generator.projectName, generator.streamInstanceName, generator.streamConnectionName),
								resources.AtlasNameToKubernetesName(),
							),
							Namespace: targetNamespace,
							Labels: map[string]string{
								secrets.TypeLabelKey: secrets.CredLabelVal,
							},
						},
						Data: map[string][]byte{secrets.UsernameField: []byte("admin"), secrets.PasswordField: []byte("")},
					},
					secret,
				)
			}
		}
	})
}

func referenceTeam(name, namespace string, users []akov2.TeamUser, projectName string, labels map[string]string) *akov2.AtlasTeam {
	dictionary := resources.AtlasNameToKubernetesName()

	return &akov2.AtlasTeam{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasTeam",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-team-%s", projectName, name), dictionary),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: akov2.TeamSpec{
			Name:      name,
			Usernames: users,
		},
		Status: akov2status.TeamStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}
}

func checkProject(t *testing.T, output []runtime.Object, expected *akov2.AtlasProject) {
	t.Helper()
	found := false
	var p *akov2.AtlasProject
	var ok bool
	for i := range output {
		p, ok = output[i].(*akov2.AtlasProject)
		if ok {
			found = true
			break
		}
	}
	require.True(t, found, "AtlasProject is not found in results")

	// secretref names are randomly generated so we can't determine those in forehand
	expected.Spec.EncryptionAtRest.AwsKms = p.Spec.EncryptionAtRest.AwsKms
	expected.Spec.EncryptionAtRest.GoogleCloudKms = p.Spec.EncryptionAtRest.GoogleCloudKms
	expected.Spec.EncryptionAtRest.AzureKeyVault = p.Spec.EncryptionAtRest.AzureKeyVault

	for i := range p.Spec.AlertConfigurations {
		alertConfig := &p.Spec.AlertConfigurations[i]
		for j := range alertConfig.Notifications {
			expected.Spec.AlertConfigurations[i].Notifications[j].APITokenRef = p.Spec.AlertConfigurations[i].Notifications[j].APITokenRef
			expected.Spec.AlertConfigurations[i].Notifications[j].DatadogAPIKeyRef = p.Spec.AlertConfigurations[i].Notifications[j].DatadogAPIKeyRef
			expected.Spec.AlertConfigurations[i].Notifications[j].OpsGenieAPIKeyRef = p.Spec.AlertConfigurations[i].Notifications[j].OpsGenieAPIKeyRef
			expected.Spec.AlertConfigurations[i].Notifications[j].ServiceKeyRef = p.Spec.AlertConfigurations[i].Notifications[j].ServiceKeyRef
			expected.Spec.AlertConfigurations[i].Notifications[j].VictorOpsSecretRef = p.Spec.AlertConfigurations[i].Notifications[j].VictorOpsSecretRef
		}
		for k := range alertConfig.Matchers {
			expected.Spec.AlertConfigurations[i].Matchers[k].FieldName = p.Spec.AlertConfigurations[i].Matchers[k].FieldName
			expected.Spec.AlertConfigurations[i].Matchers[k].Operator = p.Spec.AlertConfigurations[i].Matchers[k].Operator
			expected.Spec.AlertConfigurations[i].Matchers[k].Value = p.Spec.AlertConfigurations[i].Matchers[k].Value
		}
	}

	assert.Equal(t, expected, p)
}

func referenceProject(name, namespace string, labels map[string]string) *akov2.AtlasProject {
	dictionary := resources.AtlasNameToKubernetesName()
	return &akov2.AtlasProject{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasProject",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.NormalizeAtlasName(name, dictionary),
			Namespace: namespace,
			Labels:    labels,
		},
		Status: akov2status.AtlasProjectStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
		Spec: akov2.AtlasProjectSpec{
			Name: name,
			ConnectionSecret: &akov2common.ResourceRefNamespaced{
				Name: resources.NormalizeAtlasName(name+"-credentials", dictionary),
			},
			Settings: &akov2.ProjectSettings{
				IsCollectDatabaseSpecificsStatisticsEnabled: pointer.Get(true),
				IsDataExplorerEnabled:                       pointer.Get(true),
				IsPerformanceAdvisorEnabled:                 pointer.Get(true),
				IsRealtimePerformancePanelEnabled:           pointer.Get(true),
				IsSchemaAdvisorEnabled:                      pointer.Get(true),
			},
			Auditing: &akov2.Auditing{
				AuditAuthorizationSuccess: false,
				Enabled:                   false,
			},
			EncryptionAtRest: &akov2.EncryptionAtRest{
				AwsKms: akov2.AwsKms{
					Enabled: pointer.Get(false),
					Valid:   pointer.Get(false),
					SecretRef: akov2common.ResourceRefNamespaced{
						Name:      resources.NormalizeAtlasName(name+"-aws-credentials", dictionary),
						Namespace: namespace,
					},
				},
				AzureKeyVault: akov2.AzureKeyVault{
					Enabled: pointer.Get(false),
					SecretRef: akov2common.ResourceRefNamespaced{
						Name:      resources.NormalizeAtlasName(name+"-azure-credentials", dictionary),
						Namespace: namespace,
					},
				},
				GoogleCloudKms: akov2.GoogleCloudKms{
					Enabled: pointer.Get(false),
					SecretRef: akov2common.ResourceRefNamespaced{
						Name:      resources.NormalizeAtlasName(name+"-gcp-credentials", dictionary),
						Namespace: namespace,
					},
				},
			},
		},
	}
}

func referenceAdvancedCluster(name, region, namespace, projectName string, labels map[string]string, mdbVersion string) *akov2.AtlasDeployment {
	dictionary := resources.AtlasNameToKubernetesName()
	return &akov2.AtlasDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasDeployment",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-%s", projectName, name), dictionary),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: akov2.AtlasDeploymentSpec{
			ProjectDualReference: akov2.ProjectDualReference{
				ProjectRef: &akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(projectName, dictionary),
					Namespace: namespace,
				},
			},
			BackupScheduleRef: akov2common.ResourceRefNamespaced{
				Namespace: targetNamespace,
				Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-%s-backupschedule", projectName, name), dictionary),
			},
			DeploymentSpec: &akov2.AdvancedDeploymentSpec{
				BackupEnabled: pointer.Get(true),
				BiConnector: &akov2.BiConnectorSpec{
					Enabled:        pointer.Get(false),
					ReadPreference: "secondary",
				},
				MongoDBMajorVersion:      mdbVersion,
				ClusterType:              string(akov2.TypeReplicaSet),
				DiskSizeGB:               nil,
				EncryptionAtRestProvider: "NONE",
				Name:                     name,
				Paused:                   pointer.Get(false),
				PitEnabled:               pointer.Get(true),
				ReplicationSpecs: []*akov2.AdvancedReplicationSpec{
					{
						NumShards: 1,
						ZoneName:  "Zone 1",
						RegionConfigs: []*akov2.AdvancedRegionConfig{
							{
								AnalyticsSpecs: &akov2.Specs{
									DiskIOPS:      pointer.Get(int64(3000)),
									EbsVolumeType: "STANDARD",
									InstanceSize:  e2eClusterTier,
									NodeCount:     pointer.Get(0),
								},
								ElectableSpecs: &akov2.Specs{
									DiskIOPS:      pointer.Get(int64(3000)),
									EbsVolumeType: "STANDARD",
									InstanceSize:  e2eClusterTier,
									NodeCount:     pointer.Get(3),
								},
								ReadOnlySpecs: &akov2.Specs{
									DiskIOPS:      pointer.Get(int64(3000)),
									EbsVolumeType: "STANDARD",
									InstanceSize:  e2eClusterTier,
									NodeCount:     pointer.Get(0),
								},
								AutoScaling: &akov2.AdvancedAutoScalingSpec{
									DiskGB: &akov2.DiskGB{
										Enabled: pointer.Get(false),
									},
									Compute: &akov2.ComputeSpec{
										Enabled:          pointer.Get(false),
										ScaleDownEnabled: pointer.Get(false),
									},
								},
								Priority:     pointer.Get(7),
								ProviderName: string(akov2provider.ProviderAWS),
								RegionName:   region,
							},
						},
					},
				},
				RootCertType:         "ISRGROOTX1",
				VersionReleaseSystem: "LTS",
			},
			ProcessArgs: &akov2.ProcessArgs{
				MinimumEnabledTLSProtocol: "TLS1_2",
				JavascriptEnabled:         pointer.Get(true),
				NoTableScan:               pointer.Get(false),
			},
		},
		Status: akov2status.AtlasDeploymentStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}
}

func referenceFlex(name, region, namespace, projectName string, labels map[string]string) *akov2.AtlasDeployment {
	dictionary := resources.AtlasNameToKubernetesName()
	return &akov2.AtlasDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasDeployment",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-%s", projectName, name), dictionary),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: akov2.AtlasDeploymentSpec{
			ProjectDualReference: akov2.ProjectDualReference{
				ProjectRef: &akov2common.ResourceRefNamespaced{
					Name:      resources.NormalizeAtlasName(projectName, dictionary),
					Namespace: namespace,
				},
			},
			FlexSpec: &akov2.FlexSpec{
				Name: name,
				ProviderSettings: &akov2.FlexProviderSettings{
					BackingProviderName: string(akov2provider.ProviderAWS),
					RegionName:          region,
				},
			},
		},
		Status: akov2status.AtlasDeploymentStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}
}

func referenceBackupSchedule(namespace, projectName, clusterName string, labels map[string]string) *akov2.AtlasBackupSchedule {
	dictionary := resources.AtlasNameToKubernetesName()
	return &akov2.AtlasBackupSchedule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasBackupSchedule",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-%s-backupschedule", projectName, clusterName), dictionary),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: akov2.AtlasBackupScheduleSpec{
			PolicyRef: akov2common.ResourceRefNamespaced{
				Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-%s-backuppolicy", projectName, clusterName), dictionary),
				Namespace: namespace,
			},
			ReferenceHourOfDay:    1,
			ReferenceMinuteOfHour: 0,
			RestoreWindowDays:     7,
		},
	}
}

func referenceBackupPolicy(namespace, projectName, clusterName string, labels map[string]string) *akov2.AtlasBackupPolicy {
	dictionary := resources.AtlasNameToKubernetesName()
	return &akov2.AtlasBackupPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasBackupPolicy",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.NormalizeAtlasName(fmt.Sprintf("%s-%s-backuppolicy", projectName, clusterName), dictionary),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: akov2.AtlasBackupPolicySpec{
			Items: []akov2.AtlasBackupPolicyItem{
				{
					FrequencyType:     "hourly",
					FrequencyInterval: 6,
					RetentionUnit:     "days",
					RetentionValue:    7,
				},
				{
					FrequencyType:     "daily",
					FrequencyInterval: 1,
					RetentionUnit:     "days",
					RetentionValue:    7,
				},
				{
					FrequencyType:     "weekly",
					FrequencyInterval: 6,
					RetentionUnit:     "weeks",
					RetentionValue:    4,
				},
				{
					FrequencyType:     "monthly",
					FrequencyInterval: 40,
					RetentionUnit:     "months",
					RetentionValue:    12,
				},
				{
					FrequencyType:     "yearly",
					FrequencyInterval: 12,
					RetentionUnit:     "years",
					RetentionValue:    1,
				},
			},
		},
	}
}

func checkClustersData(t *testing.T, deployments []*akov2.AtlasDeployment, clusterNames []string, region, namespace, projectName, mDBVersion string) {
	t.Helper()
	assert.Len(t, deployments, len(clusterNames))
	var entries []string
	for _, deployment := range deployments {
		if deployment.Spec.FlexSpec != nil {
			if ok := slices.Contains(clusterNames, deployment.Spec.FlexSpec.Name); ok {
				name := deployment.Spec.FlexSpec.Name
				expectedDeployment := referenceFlex(name, region, namespace, projectName, expectedLabels)
				assert.Equal(t, expectedDeployment, deployment)
				entries = append(entries, name)
			}
		} else if deployment.Spec.DeploymentSpec != nil {
			if ok := slices.Contains(clusterNames, deployment.Spec.DeploymentSpec.Name); ok {
				name := deployment.Spec.DeploymentSpec.Name
				expectedDeployment := referenceAdvancedCluster(name, region, namespace, projectName, expectedLabels, mDBVersion)
				assert.Equal(t, expectedDeployment, deployment)
				entries = append(entries, name)
			}
		}
	}
	assert.Len(t, entries, len(clusterNames))
	assert.ElementsMatch(t, clusterNames, entries)
}

// TODO: add tests for project auditing and encryption at rest

func TestKubernetesConfigGenerate_ClustersWithBackup(t *testing.T) {
	n, err := RandInt(255)
	require.NoError(t, err)
	g := newAtlasE2ETestGenerator(t)
	g.enableBackup = true
	g.generateProject(fmt.Sprintf("kubernetes-%s", n))
	g.generateCluster()
	g.generateFlexCluster()

	expectedDeployment := referenceAdvancedCluster(g.clusterName, g.clusterRegion, targetNamespace, g.projectName, expectedLabels, g.mDBVer)
	expectedBackupSchedule := referenceBackupSchedule(targetNamespace, g.projectName, g.clusterName, expectedLabels)
	expectedBackupPolicy := referenceBackupPolicy(targetNamespace, g.projectName, g.clusterName, expectedLabels)

	cliPath, err := PluginBin()
	require.NoError(t, err)
	atlasCliPath, err := AtlasCLIBin()
	require.NoError(t, err)

	// always register atlas entities
	require.NoError(t, akov2.AddToScheme(scheme.Scheme))

	t.Run("Update backup schedule", func(t *testing.T) {
		cmd := exec.Command(atlasCliPath,
			backupsEntity,
			"schedule",
			"update",
			"--referenceHourOfDay",
			strconv.FormatInt(expectedBackupSchedule.Spec.ReferenceHourOfDay, 10),
			"--referenceMinuteOfHour",
			strconv.FormatInt(expectedBackupSchedule.Spec.ReferenceMinuteOfHour, 10),
			"--projectId",
			g.projectID,
			"--clusterName",
			g.clusterName)
		cmd.Env = os.Environ()
		resp, err := test.RunAndGetStdOut(cmd)
		require.NoError(t, err, string(resp))
	})

	t.Run("Generate valid resources of ONE project and ONE cluster", func(t *testing.T) {
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			g.projectID,
			"--clusterName",
			g.clusterName,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object

		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects, "result should not be empty")

		p, found := findAtlasProject(objects)
		require.True(t, found, "AtlasProject is not found in results")
		assert.Equal(t, targetNamespace, p.Namespace)
		found = false
		var deployment *akov2.AtlasDeployment
		var ok bool
		for i := range objects {
			deployment, ok = objects[i].(*akov2.AtlasDeployment)
			if ok {
				found = true
				break
			}
		}
		require.True(t, found, "AtlasDeployment is not found in results")
		assert.Equal(t, expectedDeployment, deployment)

		secret, found := findSecret(objects)
		require.True(t, found, "Secret is not found in results")
		assert.Equal(t, targetNamespace, secret.Namespace)
		schedule, found := atlasBackupSchedule(objects)
		require.True(t, found, "AtlasBackupSchedule is not found in results")
		assert.Equal(t, expectedBackupSchedule, schedule)
		policy, found := atlasBackupPolicy(objects)
		require.True(t, found, "AtlasBackupPolicy is not found in results")
		assert.Equal(t, expectedBackupPolicy, policy)
	})

	t.Run("Generate valid resources of ONE project and TWO clusters", func(t *testing.T) {
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			g.projectID,
			"--clusterName",
			g.clusterName,
			"--clusterName",
			g.flexName,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)
		p, found := findAtlasProject(objects)
		require.True(t, found, "AtlasProject is not found in results")
		assert.Equal(t, targetNamespace, p.Namespace)

		ds := atlasDeployments(objects)
		require.Len(t, ds, 2)
		checkClustersData(t, ds, []string{g.clusterName, g.flexName}, g.clusterRegion, targetNamespace, g.projectName, g.mDBVer)
		secret, found := findSecret(objects)
		require.True(t, found, "Secret is not found in results")
		assert.Equal(t, targetNamespace, secret.Namespace)
	})

	t.Run("Generate valid resources of ONE project and TWO clusters without listing clusters", func(t *testing.T) {
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			g.projectID,
			"--targetNamespace",
			targetNamespace,
			"--includeSecrets")
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects)
		p, found := findAtlasProject(objects)
		require.True(t, found, "AtlasProject is not found in results")
		assert.Equal(t, targetNamespace, p.Namespace)
		ds := atlasDeployments(objects)
		checkClustersData(t, ds, []string{g.clusterName, g.flexName}, g.clusterRegion, targetNamespace, g.projectName, g.mDBVer)
		secret, found := findSecret(objects)
		require.True(t, found, "Secret is not found in results")
		assert.Equal(t, targetNamespace, secret.Namespace)
	})
}

func atlasBackupPolicy(objects []runtime.Object) (*akov2.AtlasBackupPolicy, bool) {
	for i := range objects {
		if policy, ok := objects[i].(*akov2.AtlasBackupPolicy); ok {
			return policy, ok
		}
	}
	return nil, false
}

func TestKubernetesConfigGenerateFlexCluster(t *testing.T) {
	n, err := RandInt(255)
	require.NoError(t, err)
	g := newAtlasE2ETestGenerator(t)
	g.generateProject(fmt.Sprintf("kubernetes-%s", n))
	g.tier = e2eSharedClusterTier
	g.generateFlexCluster()

	expectedDeployment := referenceFlex(g.flexName, "US_EAST_1", targetNamespace, g.projectName, expectedLabels)

	cliPath, err := PluginBin()
	require.NoError(t, err)

	// always register atlas entities
	require.NoError(t, akov2.AddToScheme(scheme.Scheme))

	cmd := exec.Command(cliPath,
		"kubernetes",
		"config",
		"generate",
		"--projectId",
		g.projectID,
		"--targetNamespace",
		targetNamespace,
		"--includeSecrets")
	cmd.Env = os.Environ()

	resp, err := test.RunAndGetStdOut(cmd)
	t.Log(string(resp))
	require.NoError(t, err, string(resp))
	var objects []runtime.Object
	objects, err = getK8SEntities(resp)
	require.NoError(t, err, "should not fail on decode")
	require.NotEmpty(t, objects)

	p, found := findAtlasProject(objects)
	require.True(t, found, "AtlasProject is not found in results")
	assert.Equal(t, targetNamespace, p.Namespace)
	ds := atlasDeployments(objects)
	assert.Len(t, ds, 1)
	assert.Equal(t, expectedDeployment, ds[0])
	secret, found := findSecret(objects)
	require.True(t, found, "Secret is not found in results")
	assert.Equal(t, targetNamespace, secret.Namespace)
}

func atlasDeployments(objects []runtime.Object) []*akov2.AtlasDeployment {
	var ds []*akov2.AtlasDeployment
	for i := range objects {
		d, ok := objects[i].(*akov2.AtlasDeployment)
		if ok {
			ds = append(ds, d)
		}
	}
	return ds
}

func findAtlasProject(objects []runtime.Object) (*akov2.AtlasProject, bool) {
	for i := range objects {
		if p, ok := objects[i].(*akov2.AtlasProject); ok {
			return p, ok
		}
	}
	return nil, false
}

func findSecret(objects []runtime.Object) (*corev1.Secret, bool) {
	for i := range objects {
		if secret, ok := objects[i].(*corev1.Secret); ok {
			return secret, ok
		}
	}
	return nil, false
}

func atlasBackupSchedule(objects []runtime.Object) (*akov2.AtlasBackupSchedule, bool) {
	for i := range objects {
		if schedule, ok := objects[i].(*akov2.AtlasBackupSchedule); ok {
			return schedule, ok
		}
	}
	return nil, false
}

func TestKubernetesConfigGenerate_DataFederation(t *testing.T) {
	if revision, ok := os.LookupEnv("revision"); ok {
		t.Log(revision)
		t.Log(expectedLabels)
	}
	n, err := RandInt(255)
	require.NoError(t, err)
	g := newAtlasE2ETestGenerator(t)
	g.generateProject(fmt.Sprintf("kubernetes-%s", n))
	g.generateDataFederation()
	var storeNames []string
	storeNames = append(storeNames, g.dataFedName)
	g.generateDataFederation()
	storeNames = append(storeNames, g.dataFedName)
	expectedDataFederation := referenceDataFederation(storeNames[0], targetNamespace, g.projectName, expectedLabels)

	cliPath, err := PluginBin()
	require.NoError(t, err)

	// always register atlas entities
	require.NoError(t, akov2.AddToScheme(scheme.Scheme))

	t.Run("Generate valid resources of ONE project and ONE data federation", func(t *testing.T) {
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			g.projectID,
			"--dataFederationName",
			storeNames[0],
			"--targetNamespace",
			targetNamespace)
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))

		require.NoError(t, err, string(resp))

		var objects []runtime.Object

		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects, "result should not be empty")

		p, found := findAtlasProject(objects)
		require.True(t, found, "AtlasProject is not found in results")
		assert.Equal(t, targetNamespace, p.Namespace)
		var datafederation *akov2.AtlasDataFederation
		var ok bool
		for i := range objects {
			datafederation, ok = objects[i].(*akov2.AtlasDataFederation)
			if ok {
				found = true
				break
			}
		}
		require.True(t, found, "AtlasDataFederation is not found in results")
		assert.Equal(t, expectedDataFederation, datafederation)
	})

	t.Run("Generate valid resources of ONE project and TWO data federation", func(t *testing.T) {
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			g.projectID,
			"--dataFederationName",
			storeNames[0],
			"--dataFederationName",
			storeNames[1],
			"--targetNamespace",
			targetNamespace)
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects, "result should not be empty")
		p, found := findAtlasProject(objects)
		require.True(t, found, "AtlasProject is not found in results")
		assert.Equal(t, targetNamespace, p.Namespace)
		dataFeds := atlasDataFederations(objects)
		require.Len(t, dataFeds, len(storeNames))
		checkDataFederationData(t, dataFeds, storeNames, targetNamespace, g.projectName)
	})

	t.Run("Generate valid resources of ONE project and TWO data federation without listing data federation instances", func(t *testing.T) {
		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"generate",
			"--projectId",
			g.projectID,
			"--targetNamespace",
			targetNamespace)
		cmd.Env = os.Environ()

		resp, err := test.RunAndGetStdOut(cmd)
		t.Log(string(resp))
		require.NoError(t, err, string(resp))

		var objects []runtime.Object
		objects, err = getK8SEntities(resp)
		require.NoError(t, err, "should not fail on decode")
		require.NotEmpty(t, objects, "result should not be empty")
		p, found := findAtlasProject(objects)
		require.True(t, found, "AtlasProject is not found in results")
		assert.Equal(t, targetNamespace, p.Namespace)
		dataFeds := atlasDataFederations(objects)
		checkDataFederationData(t, dataFeds, storeNames, targetNamespace, g.projectName)
	})
}

func atlasDataFederations(objects []runtime.Object) []*akov2.AtlasDataFederation {
	var df []*akov2.AtlasDataFederation
	for i := range objects {
		d, ok := objects[i].(*akov2.AtlasDataFederation)
		if ok {
			df = append(df, d)
		}
	}
	return df
}

func checkDataFederationData(t *testing.T, dataFederations []*akov2.AtlasDataFederation, dataFedNames []string, namespace, projectName string) {
	t.Helper()
	assert.Len(t, dataFederations, len(dataFedNames))
	var entries []string
	for _, instance := range dataFederations {
		if ok := slices.Contains(dataFedNames, instance.Spec.Name); ok {
			name := instance.Spec.Name
			expectedDeployment := referenceDataFederation(name, namespace, projectName, expectedLabels)
			assert.Equal(t, expectedDeployment, instance)
			entries = append(entries, name)
		}
	}
	assert.ElementsMatch(t, dataFedNames, entries)
}

func TestGenerateMany(t *testing.T) {
	// always register atlas entities
	require.NoError(t, akov2.AddToScheme(scheme.Scheme))
	projectID, projectName := generateTestAtlasProject(t)

	// Test ipAccessList with pagination (25 entries)
	ipAccessList := make([]string, 25)
	for i := range ipAccessList {
		ip := fmt.Sprintf("192.0.0.%d", i)
		ipAccessList[i] = generateTestAtlasIPAccessList(t, projectID, "ip", ip)
	}

	// Test alertConfigurations with pagination (25 entries)
	alertMarkers := make([]string, 25)
	for i := range alertMarkers {
		marker := fmt.Sprintf("alert-marker-%02d-%s", i+1, randSuffix(t))
		alertMarkers[i] = generateTestAtlasAlertConfiguration(t, projectID, marker)
	}

	// Test databaseUsers with pagination (25 entries)
	dbUsers := make([]string, 25)
	for i := range dbUsers {
		user := fmt.Sprintf("dbuser-%02d-%s", i+1, randSuffix(t))
		dbUsers[i] = generateTestAtlasDatabaseUser(t, projectID, user, "pass")
	}

	// Test all integrations with pagination
	generateTestAtlasIntegration(t, projectID, datadogEntity)
	generateTestAtlasIntegration(t, projectID, opsgenieEntity)
	generateTestAtlasIntegration(t, projectID, victoropsEntity)
	generateTestAtlasIntegration(t, projectID, pagerdutyEntity)
	generateTestAtlasIntegration(t, projectID, webhookEntity)

	// Test Streams instances with pagination
	streams := make([]string, 2)
	for i := range streams {
		streams[i] = fmt.Sprintf("stream-%02d-%s", i+1, randSuffix(t))
		generateTestAtlasStreamInstance(t, projectID, streams[i])
	}

	// Test flex clusters with pagination
	flexClusters := make([]string, 2)
	for i := range flexClusters {
		flexClusters[i] = fmt.Sprintf("flex-%02d-%s", i+1, randSuffix(t))
		generateTestAtlasFlexCluster(t, projectID, flexClusters[i])
	}

	// Test Normal (M10) Atlas clusters
	advancedDeployments := make([]string, 2)
	for i := range advancedDeployments {
		advancedDeployments[i] = fmt.Sprintf("adv-deployment-%02d-%s", i+1, randSuffix(t))
		generateTestAtlasAdvancedDeployment(t, projectID, advancedDeployments[i])
	}

	cliPath, err := PluginBin()
	require.NoError(t, err)
	cmd := exec.Command(cliPath,
		"kubernetes",
		"config",
		"generate",
		"--projectId", projectID,
		"--targetNamespace", targetNamespace,
		"--independentResources")
	cmd.Env = os.Environ()

	resp, err := test.RunAndGetStdOut(cmd)
	t.Log(string(resp))
	require.NoError(t, err, string(resp))

	objects, err := getK8SEntities(resp)
	require.NoError(t, err)
	require.NotEmpty(t, objects)

	assert.NotNil(t, findTestAtlasProject(objects, projectName))

	for i, ip := range ipAccessList {
		assert.NotNil(t, findTestAtlasIPAccessList(objects, projectID, ip), "IP access list %d with ID %s not found", i+1, ip)
	}

	for i, marker := range alertMarkers {
		assert.NotNil(t, findTestAtlasAlertConfiguration(objects, projectName, marker), "Alert configuration %d with marker %s not found", i+1, marker)
	}

	for i, name := range dbUsers {
		assert.NotNil(t, findTestAtlasDatabaseUser(objects, projectID, name), "DB user %d with username %s not found", i+1, name)
	}

	for i, typ := range []string{datadogEntity, opsgenieEntity, victoropsEntity, pagerdutyEntity, webhookEntity} {
		assert.NotNil(t, findTestAtlasIntegration(objects, projectID, typ), "Integration %d of type %s not found", i+1, typ)
	}

	for i, name := range streams {
		assert.NotNil(t, findTestAtlasStreamInstance(objects, projectName, name), "Stream instance %d with name %s not found", i+1, name)
	}

	for i, name := range flexClusters {
		assert.NotNil(t, findTestAtlasFlexCluster(objects, projectID, name), "Flex cluster %d with name %s not found", i+1, name)
	}

	for i, name := range advancedDeployments {
		assert.NotNil(t, findTestAtlasAdvancedDeployment(objects, projectID, name), "Cluster %d with name %s not found", i+1, name)
	}
}

func randSuffix(t *testing.T) string {
	alpha := func() string {
		b := make([]byte, 2)
		for i := range b {
			val, err := RandInt(26)
			require.NoError(t, err)
			b[i] = byte('a' + val.Int64())
		}
		return string(b)
	}
	num, err := RandInt(1000)
	require.NoError(t, err)
	return fmt.Sprintf("%s%03d", alpha(), num.Int64())
}
