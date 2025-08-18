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

//go:build unit

package project

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/features"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/resources"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/mocks"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/pointer"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api"
	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/common"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/atlas-sdk/v20241113004/admin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestBuildIntegrations(t *testing.T) {
	projectID := "project-int-id"
	projectName := "projectName-int"
	targetNamespace := "intNamespace"
	credentialName := "int-creds"
	version := features.LatestOperatorMajorVersion

	for _, tc := range []struct {
		title               string
		atlasIntegration    []admin.ThirdPartyIntegration
		independentResource bool
		wantResources       []runtime.Object
	}{
		{
			title:            "no integrations",
			atlasIntegration: []admin.ThirdPartyIntegration{},
			wantResources:    []runtime.Object{},
		},
		{
			title: "Datadog integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:                           pointer.Get("integration-id"),
					Type:                         pointer.Get("DATADOG"),
					ApiKey:                       pointer.Get("fake-api-key"),
					Region:                       pointer.Get("US"),
					SendCollectionLatencyMetrics: pointer.Get(true),
					SendDatabaseMetrics:          pointer.Get(false),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-datadog-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						Type: "DATADOG",
						Datadog: &akov2.DatadogIntegration{
							APIKeySecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-datadog-integration-secret",
							},
							Region:                       "US",
							SendCollectionLatencyMetrics: pointer.Get("enabled"),
							SendDatabaseMetrics:          pointer.Get("disabled"),
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-datadog-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"apiKey": ([]byte)("fake-api-key"),
					},
				},
			},
			independentResource: false,
		},
		{
			title: "Microsoft Teams integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:   pointer.Get("integration-id"),
					Type: pointer.Get("MICROSOFT_TEAMS"),
					Url:  pointer.Get("fake-url"),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-microsoft-teams-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						Type: "MICROSOFT_TEAMS",
						MicrosoftTeams: &akov2.MicrosoftTeamsIntegration{
							URLSecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-microsoft-teams-integration-secret",
							},
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-microsoft-teams-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"url": ([]byte)("fake-url"),
					},
				},
			},
			independentResource: true,
		},
		{
			title: "New Relic integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:         pointer.Get("integration-id"),
					Type:       pointer.Get("NEW_RELIC"),
					AccountId:  pointer.Get("fake-account-id"),
					LicenseKey: pointer.Get("fake-license-key"),
					ReadToken:  pointer.Get("fake-read-token"),
					WriteToken: pointer.Get("fake-write-token"),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-new-relic-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						Type: "NEW_RELIC",
						NewRelic: &akov2.NewRelicIntegration{
							CredentialsSecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-new-relic-integration-secret",
							},
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-new-relic-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"accountId":  ([]byte)("fake-account-id"),
						"licenseKey": ([]byte)("fake-license-key"),
						"readToken":  ([]byte)("fake-read-token"),
						"writeToken": ([]byte)("fake-write-token"),
					},
				},
			},
			independentResource: false,
		},
		{
			title: "Ops Genie integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:     pointer.Get("integration-id"),
					Type:   pointer.Get("OPS_GENIE"),
					Region: pointer.Get("EU"),
					ApiKey: pointer.Get("fake-api-key"),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-ops-genie-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						Type: "OPS_GENIE",
						OpsGenie: &akov2.OpsGenieIntegration{
							APIKeySecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-ops-genie-integration-secret",
							},
							Region: "EU",
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-ops-genie-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"apiKey": ([]byte)("fake-api-key"),
					},
				},
			},
			independentResource: true,
		},
		{
			title: "Pager duty integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:         pointer.Get("integration-id"),
					Type:       pointer.Get("PAGER_DUTY"),
					ServiceKey: pointer.Get("fake-service-key"),
					Region:     pointer.Get("US"),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-pager-duty-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						Type: "PAGER_DUTY",
						PagerDuty: &akov2.PagerDutyIntegration{
							ServiceKeySecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-pager-duty-integration-secret",
							},
							Region: "US",
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-pager-duty-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"serviceKey": ([]byte)("fake-service-key"),
					},
				},
			},
			independentResource: false,
		},
		{
			title: "Prometheus integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:               pointer.Get("integration-id"),
					Type:             pointer.Get("PROMETHEUS"),
					Username:         pointer.Get("fake-username"),
					Password:         pointer.Get("fake-password"),
					Enabled:          pointer.Get(true),
					ServiceDiscovery: pointer.Get("http"),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-prometheus-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						Type: "PROMETHEUS",
						Prometheus: &akov2.PrometheusIntegration{
							PrometheusCredentialsSecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-prometheus-integration-secret",
							},
							Enabled:          pointer.Get("enabled"),
							ServiceDiscovery: "http",
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-prometheus-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"username": ([]byte)("fake-username"),
						"password": ([]byte)("fake-password"),
					},
				},
			},
			independentResource: true,
		},
		{
			title: "Slack integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:          pointer.Get("integration-id"),
					Type:        pointer.Get("SLACK"),
					ApiToken:    pointer.Get("fake-api-token"),
					ChannelName: pointer.Get("a-channel"),
					TeamName:    pointer.Get("a-team"),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-slack-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						Type: "SLACK",
						Slack: &akov2.SlackIntegration{
							APITokenSecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-slack-integration-secret",
							},
							ChannelName: "a-channel",
							TeamName:    "a-team",
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-slack-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"apiToken": ([]byte)("fake-api-token"),
					},
				},
			},
			independentResource: false,
		},
		{
			title: "Victor Ops integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:         pointer.Get("integration-id"),
					Type:       pointer.Get("VICTOR_OPS"),
					ApiKey:     pointer.Get("fake-api-key"),
					RoutingKey: pointer.Get("routing-key"),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-victor-ops-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						Type: "VICTOR_OPS",
						VictorOps: &akov2.VictorOpsIntegration{
							APIKeySecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-victor-ops-integration-secret",
							},
							RoutingKey: "routing-key",
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-victor-ops-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"apiKey": ([]byte)("fake-api-key"),
					},
				},
			},
			independentResource: true,
		},
		{
			title: "Webhook integration properly exported",
			atlasIntegration: []admin.ThirdPartyIntegration{
				{
					Id:     pointer.Get("integration-id"),
					Type:   pointer.Get("WEBHOOK"),
					Url:    pointer.Get("fake-url"),
					Secret: pointer.Get("fake-secret"),
				},
			},
			wantResources: []runtime.Object{
				&akov2.AtlasThirdPartyIntegration{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasThirdPartyIntegration",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-webhook-integration",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasThirdPartyIntegrationSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						Type: "WEBHOOK",
						Webhook: &akov2.WebhookIntegration{
							URLSecretRef: api.LocalObjectReference{
								Name: strings.ToLower(projectName) + "-webhook-integration-secret",
							},
						},
					},
					Status: status.AtlasThirdPartyIntegrationStatus{
						UnifiedStatus: status.UnifiedStatus{},
						ID:            "integration-id",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-webhook-integration-secret",
						Namespace: targetNamespace,
						Labels: map[string]string{
							"atlas.mongodb.com/project-id":   "project-int-id",
							"atlas.mongodb.com/project-name": "projectname-int",
							"atlas.mongodb.com/type":         "credentials",
						},
					},
					Data: map[string][]byte{
						"url":    ([]byte)("fake-url"),
						"secret": ([]byte)("fake-secret"),
					},
				},
			},
			independentResource: false,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			ctl := gomock.NewController(t)
			intStore := mocks.NewMockIntegrationLister(ctl)
			dictionary := resources.AtlasNameToKubernetesName()

			intStore.EXPECT().Integrations(projectID).Return(tc.atlasIntegration, nil)

			peerings, err := BuildThirdPartyIntegrations(
				intStore,
				ThirdPartyIntegrationRequest{
					ProjectName:         projectName,
					ProjectID:           projectID,
					TargetNamespace:     targetNamespace,
					Version:             version,
					Credentials:         credentialName,
					IndependentResource: tc.independentResource,
					Dictionary:          dictionary,
				},
			)
			require.NoError(t, err)
			assert.Equal(t, tc.wantResources, peerings)
		})
	}
}
