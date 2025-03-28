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

//go:build e2e || apply

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/features"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/resources"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/pointer"
	akoapi "github.com/mongodb/mongodb-atlas-kubernetes/v2/api"
	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1"
	akov2common "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestKubernetesConfigApply(t *testing.T) {
	cliPath, err := PluginBin()
	require.NoError(t, err)

	t.Run("should fail to apply resources when namespace do not exist", func(t *testing.T) {
		operator := setupCluster(t, "k8s-config-apply-wrong-ns", defaultOperatorNamespace)
		err = operator.installOperator(defaultOperatorNamespace, features.LatestOperatorMajorVersion)
		require.NoError(t, err)

		g := newAtlasE2ETestGenerator(t)
		g.generateProject("k8sConfigApplyWrongNs")

		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"apply",
			"--targetNamespace",
			"a-wrong-namespace",
			"--projectId",
			g.projectID)
		cmd.Env = os.Environ()
		resp, err := cmd.CombinedOutput()
		require.Error(t, err, string(resp))
		assert.Contains(t, string(resp), "Error: namespaces \"a-wrong-namespace\" not found\n")
	})

	t.Run("should fail to apply resources when unable to autodetect parameters", func(t *testing.T) {
		g := newAtlasE2ETestGenerator(t)

		setupCluster(t, "k8s-config-apply-no-auto-detect", defaultOperatorNamespace)

		g.generateProject("k8sConfigApplyNoAutoDetect")

		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"apply",
			"--targetNamespace", "e2e-autodetect-parameters",
			"--projectId", g.projectID)
		cmd.Env = os.Environ()
		resp, err := cmd.CombinedOutput()
		require.Error(t, err, string(resp))
		assert.Contains(t, string(resp), "Error: unable to auto detect params: couldn't find an operator installed in any accessible namespace\n")
	})

	t.Run("should fail to apply resources when unable to autodetect operator version", func(t *testing.T) {
		g := newAtlasE2ETestGenerator(t)

		operator := setupCluster(t, "k8s-config-apply-fail-version", defaultOperatorNamespace)
		err = operator.installOperator(defaultOperatorNamespace, features.LatestOperatorMajorVersion)
		require.NoError(t, err)

		operator.emulateCertifiedOperator()
		g.t.Cleanup(func() {
			operator.restoreOperatorImage()
		})

		g.generateProject("k8sConfigApplyFailVersion")

		e2eNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "e2e-autodetect-operator-version",
			},
		}
		require.NoError(t, operator.createK8sObject(e2eNamespace))
		g.t.Cleanup(func() {
			require.NoError(t, operator.deleteK8sObject(e2eNamespace))
		})

		cmd := exec.Command(cliPath,
			"kubernetes",
			"config",
			"apply",
			"--targetNamespace", defaultOperatorNamespace,
			"--projectId", g.projectID)
		cmd.Env = os.Environ()
		resp, err := cmd.CombinedOutput()
		require.Error(t, err, string(resp))
		assert.Contains(t, string(resp), "Error: unable to auto detect operator version. you should explicitly set operator version if you are running an openshift certified installation\n")
	})
}

func TestKubernetesConfigApplyOptions(t *testing.T) {
	cliPath, err := PluginBin()
	require.NoError(t, err)

	operator := setupCluster(t, "k8s-config-apply", defaultOperatorNamespace)
	err = operator.installOperator(defaultOperatorNamespace, features.LatestOperatorMajorVersion)
	require.NoError(t, err)

	// we don't want the operator to do reconcile and avoid conflict with cli actions
	operator.stopOperator()

	g := setupAtlasResources(t)
	g.generateDataFederation()
	var storeNames []string
	storeNames = append(storeNames, g.dataFedName)
	g.generateDataFederation()
	storeNames = append(storeNames, g.dataFedName)

	tests := map[string]struct {
		flags                   string
		independentResource     bool
		dataFederationResources map[string]bool
	}{
		"export and apply atlas resource to kubernetes cluster": {
			dataFederationResources: map[string]bool{storeNames[0]: true, storeNames[1]: true},
		},
		"export and apply selected data federation": {
			flags:                   "--dataFederationName " + storeNames[0],
			dataFederationResources: map[string]bool{storeNames[0]: true, storeNames[1]: false},
		},
		"export and apply as independent resources to kubernetes cluster": {
			flags:                   "--independentResources",
			independentResource:     true,
			dataFederationResources: map[string]bool{storeNames[0]: true, storeNames[1]: true},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			e2eNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "e2e-export-atlas-resource-" + uuid.New().String()[0:5],
				},
			}
			g.Logf("add target namespace %s", e2eNamespace.Name)
			require.NoError(t, operator.createK8sObject(e2eNamespace))
			g.t.Cleanup(func() {
				require.NoError(t, operator.deleteK8sObject(e2eNamespace))
			})

			flags := strings.TrimSpace(fmt.Sprintf("kubernetes config apply --targetNamespace %s --projectId %s %s", e2eNamespace.Name, g.projectID, tc.flags))
			g.Logf("executing: %s", flags)
			cmd := exec.Command(cliPath, strings.Split(flags, " ")...)
			cmd.Env = os.Environ()
			resp, err := cmd.CombinedOutput()
			require.NoError(t, err, string(resp))
			t.Log(string(resp))

			akoProject := akov2.AtlasProject{}
			require.NoError(
				t,
				operator.getK8sObject(
					client.ObjectKey{Name: prepareK8sName(g.projectName), Namespace: e2eNamespace.Name},
					&akoProject,
					true,
				),
			)
			assert.NotEmpty(t, akoProject.Spec.AlertConfigurations)
			akoProject.Spec.AlertConfigurations = nil
			assert.Equal(t, referenceExportedProject(g.projectName, g.teamName, &akoProject).Spec, akoProject.Spec)

			// Assert Database User
			akoDBUser := akov2.AtlasDatabaseUser{}
			require.NoError(
				t,
				operator.getK8sObject(
					client.ObjectKey{Name: prepareK8sName(fmt.Sprintf("%s-%s", g.projectName, g.dbUser)), Namespace: e2eNamespace.Name},
					&akoDBUser,
					true,
				),
			)
			assert.Equal(t, referenceExportedDBUser(g, e2eNamespace.Name, tc.independentResource).Spec, akoDBUser.Spec)

			// Assert Team
			akoTeam := akov2.AtlasTeam{}
			require.NoError(
				t,
				operator.getK8sObject(
					client.ObjectKey{Name: prepareK8sName(fmt.Sprintf("%s-team-%s", g.projectName, g.teamName)), Namespace: e2eNamespace.Name},
					&akoTeam,
					true,
				),
			)
			assert.Equal(t, referenceExportedTeam(g.teamName, g.teamUser).Spec, akoTeam.Spec)

			// Assert Backup Policy
			akoBkpPolicy := akov2.AtlasBackupPolicy{}
			require.NoError(
				t,
				operator.getK8sObject(
					client.ObjectKey{Name: prepareK8sName(fmt.Sprintf("%s-%s-backuppolicy", g.projectName, g.clusterName)), Namespace: e2eNamespace.Name},
					&akoBkpPolicy,
					true,
				),
			)
			assert.Equal(t, referenceExportedBackupPolicy().Spec, akoBkpPolicy.Spec)

			// Assert Backup Schedule
			akoBkpSchedule := akov2.AtlasBackupSchedule{}
			require.NoError(
				t,
				operator.getK8sObject(
					client.ObjectKey{Name: prepareK8sName(fmt.Sprintf("%s-%s-backupschedule", g.projectName, g.clusterName)), Namespace: e2eNamespace.Name},
					&akoBkpSchedule,
					true,
				),
			)
			assert.Equal(
				t,
				referenceExportedBackupSchedule(g.projectName, g.clusterName, e2eNamespace.Name, akoBkpSchedule.Spec.ReferenceHourOfDay, akoBkpSchedule.Spec.ReferenceMinuteOfHour).Spec,
				akoBkpSchedule.Spec,
			)

			// Assert Deployment
			akoDeployment := akov2.AtlasDeployment{}
			require.NoError(
				t,
				operator.getK8sObject(
					client.ObjectKey{Name: prepareK8sName(fmt.Sprintf("%s-%s", g.projectName, g.clusterName)), Namespace: e2eNamespace.Name},
					&akoDeployment,
					true,
				),
			)
			assert.Equal(t, referenceExportedDeployment(g, e2eNamespace.Name, g.mDBVer, tc.independentResource).Spec, akoDeployment.Spec)

			// Assert Data Federation
			for dataFedName, exported := range tc.dataFederationResources {
				switch exported {
				case true:
					akoDataFed := akov2.AtlasDataFederation{}
					require.NoError(
						t,
						operator.getK8sObject(
							client.ObjectKey{Name: prepareK8sName(fmt.Sprintf("%s-%s", g.projectName, dataFedName)), Namespace: e2eNamespace.Name},
							&akoDataFed,
							true,
						),
					)
					assert.Equal(t, referenceDataFederation(dataFedName, e2eNamespace.Name, g.projectName, nil).Spec, akoDataFed.Spec)
				case false:
					akoDataFed := akov2.AtlasDataFederation{}
					require.Error(
						t,
						operator.getK8sObject(
							client.ObjectKey{Name: prepareK8sName(fmt.Sprintf("%s-%s", g.projectName, dataFedName)), Namespace: e2eNamespace.Name},
							&akoDataFed,
							true,
						),
					)
				}
			}
		})
	}
}

func setupAtlasResources(t *testing.T) *atlasE2ETestGenerator {
	t.Helper()

	g := newAtlasE2ETestGeneratorWithBackup(t)
	g.generateProject("k8sConfigApply")
	g.generateCluster()
	g.generateTeam("k8sConfigApply")
	g.generateDBUser("k8sConfigApply")

	cliPath, err := AtlasCLIBin()
	require.NoError(t, err)

	cmd := exec.Command(cliPath,
		projectsEntity,
		teamsEntity,
		"add",
		g.teamID,
		"--role",
		"GROUP_OWNER",
		"--projectId",
		g.projectID,
		"-o=json")
	cmd.Env = os.Environ()
	resp, err := cmd.CombinedOutput()
	require.NoError(t, err, string(resp))
	g.t.Cleanup(func() {
		deleteTeamFromProject(g.t, cliPath, g.projectID, g.teamID)
	})

	return g
}

const credSuffix = "-credentials"

func referenceExportedProject(projectName, teamName string, expectedProject *akov2.AtlasProject) *akov2.AtlasProject {
	return &akov2.AtlasProject{
		Spec: akov2.AtlasProjectSpec{
			Name: projectName,
			ConnectionSecret: &akov2common.ResourceRefNamespaced{
				Name: prepareK8sName(projectName + credSuffix),
			},
			WithDefaultAlertsSettings: true,
			Settings: &akov2.ProjectSettings{
				IsCollectDatabaseSpecificsStatisticsEnabled: pointer.Get(true),
				IsDataExplorerEnabled:                       pointer.Get(true),
				IsPerformanceAdvisorEnabled:                 pointer.Get(true),
				IsRealtimePerformancePanelEnabled:           pointer.Get(true),
				IsSchemaAdvisorEnabled:                      pointer.Get(true),
			},
			EncryptionAtRest: &akov2.EncryptionAtRest{
				AwsKms: akov2.AwsKms{
					Enabled: pointer.Get(false),
					Valid:   pointer.Get(false),
					SecretRef: akov2common.ResourceRefNamespaced{
						Name:      expectedProject.Spec.EncryptionAtRest.AwsKms.SecretRef.Name,
						Namespace: expectedProject.Spec.EncryptionAtRest.AwsKms.SecretRef.Namespace,
					},
				},
				AzureKeyVault: akov2.AzureKeyVault{
					Enabled: pointer.Get(false),
					SecretRef: akov2common.ResourceRefNamespaced{
						Name:      expectedProject.Spec.EncryptionAtRest.AzureKeyVault.SecretRef.Name,
						Namespace: expectedProject.Spec.EncryptionAtRest.AzureKeyVault.SecretRef.Namespace,
					},
				},
				GoogleCloudKms: akov2.GoogleCloudKms{
					Enabled: pointer.Get(false),
					SecretRef: akov2common.ResourceRefNamespaced{
						Name:      expectedProject.Spec.EncryptionAtRest.GoogleCloudKms.SecretRef.Name,
						Namespace: expectedProject.Spec.EncryptionAtRest.GoogleCloudKms.SecretRef.Namespace,
					},
				},
			},
			Auditing: &akov2.Auditing{
				AuditAuthorizationSuccess: false,
				Enabled:                   false,
			},
			Teams: []akov2.Team{
				{
					TeamRef: akov2common.ResourceRefNamespaced{
						Namespace: expectedProject.Namespace,
						Name:      prepareK8sName(fmt.Sprintf("%s-team-%s", projectName, teamName)),
					},
					Roles: []akov2.TeamRole{
						"GROUP_OWNER",
					},
				},
			},
			RegionUsageRestrictions: "NONE",
		},
	}
}

func referenceExportedDBUser(generator *atlasE2ETestGenerator, namespace string, independent bool) *akov2.AtlasDatabaseUser {
	r := &akov2.AtlasDatabaseUser{
		Spec: akov2.AtlasDatabaseUserSpec{
			Roles: []akov2.RoleSpec{
				{
					RoleName:     "readAnyDatabase",
					DatabaseName: "admin",
				},
			},
			Username:     generator.dbUser,
			OIDCAuthType: "NONE",
			AWSIAMType:   "NONE",
			X509Type:     "MANAGED",
			DatabaseName: "$external",
		},
	}

	if independent {
		r.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ExternalProjectRef: &akov2.ExternalProjectReference{
				ID: generator.projectID,
			},
			ConnectionSecret: &akoapi.LocalObjectReference{
				Name: resources.NormalizeAtlasName(strings.ToLower(generator.projectName)+"-credentials", resources.AtlasNameToKubernetesName()),
			},
		}
	} else {
		r.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ProjectRef: &akov2common.ResourceRefNamespaced{
				Name:      strings.ToLower(generator.projectName),
				Namespace: namespace,
			},
		}
	}

	return r
}

func referenceExportedTeam(teamName, username string) *akov2.AtlasTeam {
	return &akov2.AtlasTeam{
		Spec: akov2.TeamSpec{
			Name: teamName,
			Usernames: []akov2.TeamUser{
				akov2.TeamUser(username),
			},
		},
	}
}

func referenceExportedBackupSchedule(projectName, clusterName, namespace string, refHour, refMin int64) *akov2.AtlasBackupSchedule {
	return &akov2.AtlasBackupSchedule{
		Spec: akov2.AtlasBackupScheduleSpec{
			PolicyRef: akov2common.ResourceRefNamespaced{
				Name:      prepareK8sName(fmt.Sprintf("%s-%s-backuppolicy", projectName, clusterName)),
				Namespace: namespace,
			},
			AutoExportEnabled:     false,
			ReferenceHourOfDay:    refHour,
			ReferenceMinuteOfHour: refMin,
			RestoreWindowDays:     7,
		},
	}
}

func referenceExportedBackupPolicy() *akov2.AtlasBackupPolicy {
	return &akov2.AtlasBackupPolicy{
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

func referenceExportedDeployment(generator *atlasE2ETestGenerator, namespace, mdbVersion string, independent bool) *akov2.AtlasDeployment {
	r := &akov2.AtlasDeployment{
		Spec: akov2.AtlasDeploymentSpec{
			BackupScheduleRef: akov2common.ResourceRefNamespaced{
				Name:      prepareK8sName(fmt.Sprintf("%s-%s-backupschedule", generator.projectName, generator.clusterName)),
				Namespace: namespace,
			},
			DeploymentSpec: &akov2.AdvancedDeploymentSpec{
				Name:          generator.clusterName,
				BackupEnabled: pointer.Get(true),
				BiConnector: &akov2.BiConnectorSpec{
					Enabled:        pointer.Get(false),
					ReadPreference: "secondary",
				},
				MongoDBMajorVersion:      mdbVersion,
				ClusterType:              "REPLICASET",
				EncryptionAtRestProvider: "NONE",
				Paused:                   pointer.Get(false),
				PitEnabled:               pointer.Get(true),
				ReplicationSpecs: []*akov2.AdvancedReplicationSpec{
					{
						NumShards: 1,
						ZoneName:  "Zone 1",
						RegionConfigs: []*akov2.AdvancedRegionConfig{
							{
								AnalyticsSpecs: &akov2.Specs{
									DiskIOPS:      pointer.Get[int64](3000),
									EbsVolumeType: "STANDARD",
									InstanceSize:  "M10",
									NodeCount:     pointer.Get(0),
								},
								ElectableSpecs: &akov2.Specs{
									DiskIOPS:      pointer.Get[int64](3000),
									EbsVolumeType: "STANDARD",
									InstanceSize:  "M10",
									NodeCount:     pointer.Get(3),
								},
								ReadOnlySpecs: &akov2.Specs{
									DiskIOPS:      pointer.Get[int64](3000),
									EbsVolumeType: "STANDARD",
									InstanceSize:  "M10",
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
								ProviderName: "AWS",
								RegionName:   "US_EAST_1",
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
	}

	if independent {
		r.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ExternalProjectRef: &akov2.ExternalProjectReference{
				ID: generator.projectID,
			},
			ConnectionSecret: &akoapi.LocalObjectReference{
				Name: resources.NormalizeAtlasName(strings.ToLower(generator.projectName)+"-credentials", resources.AtlasNameToKubernetesName()),
			},
		}
	} else {
		r.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ProjectRef: &akov2common.ResourceRefNamespaced{
				Name:      strings.ToLower(generator.projectName),
				Namespace: namespace,
			},
		}
	}

	return r
}

func deleteTeamFromProject(t *testing.T, cliPath, projectID, teamID string) {
	t.Helper()

	cmd := exec.Command(cliPath,
		projectsEntity,
		teamsEntity,
		"delete",
		teamID,
		"--projectId",
		projectID,
		"--force")
	cmd.Env = os.Environ()
	resp, err := cmd.CombinedOutput()
	require.NoError(t, err, string(resp))
}
