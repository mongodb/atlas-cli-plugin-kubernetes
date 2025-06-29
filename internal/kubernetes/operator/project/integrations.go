// Copyright 2025 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package project

import (
	"fmt"
	"strings"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/features"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/resources"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/secrets"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/pointer"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	akoapi "github.com/mongodb/mongodb-atlas-kubernetes/v2/api"
	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1"
	akov2common "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/common"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/status"
	"go.mongodb.org/atlas-sdk/v20241113004/admin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ThirdPartyIntegrationRequest struct {
	ProjectName         string
	ProjectID           string
	TargetNamespace     string
	Version             string
	Credentials         string
	IndependentResource bool
	Dictionary          map[string]string
}

func BuildThirdPartyIntegrations(
	provider store.IntegrationLister,
	request ThirdPartyIntegrationRequest,
) ([]runtime.Object, error) {
	atlasIntegrations, err := provider.AllIntegrations(request.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list third party integrations from Atlas: %w", err)
	}

	kubeIntegrations := make([]runtime.Object, 0, len(atlasIntegrations))
	for _, atlasIntegration := range atlasIntegrations {
		kubeIntegrations = append(kubeIntegrations, convertIntegrationToKubernetes(request, &atlasIntegration)...)
	}
	return kubeIntegrations, nil
}

func convertIntegrationToKubernetes(request ThirdPartyIntegrationRequest, atlasIntegration *admin.ThirdPartyIntegration) []runtime.Object {
	results := make([]runtime.Object, 0, 2)
	resource := akov2.AtlasThirdPartyIntegration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasThirdPartyIntegration",
			APIVersion: fmt.Sprintf("%s/%s", akov2.GroupVersion.Group, akov2.GroupVersion.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      integrationResourceName(request, atlasIntegration),
			Namespace: request.TargetNamespace,
			Labels: map[string]string{
				features.ResourceVersion: request.Version,
			},
		},
		Spec: akov2.AtlasThirdPartyIntegrationSpec{
			Type: atlasIntegration.GetType(),
		},
		Status: status.AtlasThirdPartyIntegrationStatus{
			ID: atlasIntegration.GetId(),
		},
	}
	results = append(results, &resource)
	secretName := integrationResourceSecret(request, atlasIntegration)
	switch atlasIntegration.GetType() {
	case "DATADOG":
		results = append(results, newIntegrationSecret(request, secretName, "apiKey", atlasIntegration.GetApiKey()))
		resource.Spec.Datadog = &akov2.DatadogIntegration{
			APIKeySecretRef:              akoapi.LocalObjectReference{Name: secretName},
			Region:                       atlasIntegration.GetRegion(),
			SendCollectionLatencyMetrics: convertEnabledString(atlasIntegration.GetSendCollectionLatencyMetrics()),
			SendDatabaseMetrics:          convertEnabledString(atlasIntegration.GetSendDatabaseMetrics()),
		}
	case "MICROSOFT_TEAMS":
		results = append(results, newIntegrationSecret(request, secretName, "url", atlasIntegration.GetUrl()))
		resource.Spec.MicrosoftTeams = &akov2.MicrosoftTeamsIntegration{
			URLSecretRef: akoapi.LocalObjectReference{Name: secretName},
		}
	case "NEW_RELIC":
		results = append(results, newIntegrationRawSecret(request, secretName, map[string][]byte{
			"accountId":  []byte(atlasIntegration.GetAccountId()),
			"licenseKey": []byte(atlasIntegration.GetLicenseKey()),
			"readToken":  []byte(atlasIntegration.GetReadToken()),
			"writeToken": []byte(atlasIntegration.GetWriteToken()),
		}))
		resource.Spec.NewRelic = &akov2.NewRelicIntegration{
			CredentialsSecretRef: akoapi.LocalObjectReference{Name: secretName},
		}
	case "OPS_GENIE":
		results = append(results, newIntegrationSecret(request, secretName, "apiKey", atlasIntegration.GetApiKey()))
		resource.Spec.OpsGenie = &akov2.OpsGenieIntegration{
			APIKeySecretRef: akoapi.LocalObjectReference{Name: secretName},
			Region:          atlasIntegration.GetRegion(),
		}
	case "PAGER_DUTY":
		results = append(results, newIntegrationSecret(
			request, secretName, "serviceKey", atlasIntegration.GetServiceKey()))
		resource.Spec.PagerDuty = &akov2.PagerDutyIntegration{
			ServiceKeySecretRef: akoapi.LocalObjectReference{Name: secretName},
			Region:              atlasIntegration.GetRegion(),
		}
	case "PROMETHEUS":
		results = append(results, newIntegrationRawSecret(request, secretName, map[string][]byte{
			"username": []byte(atlasIntegration.GetUsername()),
			"password": []byte(atlasIntegration.GetPassword()),
		}))
		resource.Spec.Prometheus = &akov2.PrometheusIntegration{
			PrometheusCredentialsSecretRef: akoapi.LocalObjectReference{Name: secretName},
			Enabled:                        convertEnabledString(atlasIntegration.GetEnabled()),
			ServiceDiscovery:               atlasIntegration.GetServiceDiscovery(),
		}
	case "SLACK":
		results = append(results, newIntegrationSecret(
			request, secretName, "apiToken", atlasIntegration.GetApiToken()))
		resource.Spec.Slack = &akov2.SlackIntegration{
			APITokenSecretRef: akoapi.LocalObjectReference{Name: secretName},
			ChannelName:       atlasIntegration.GetChannelName(),
			TeamName:          atlasIntegration.GetTeamName(),
		}
	case "VICTOR_OPS":
		results = append(results, newIntegrationSecret(
			request, secretName, "apiKey", atlasIntegration.GetApiKey()))
		resource.Spec.VictorOps = &akov2.VictorOpsIntegration{
			APIKeySecretRef: akoapi.LocalObjectReference{Name: secretName},
			RoutingKey:      atlasIntegration.GetRoutingKey(),
		}
	case "WEBHOOK":
		results = append(results, newIntegrationRawSecret(request, secretName, map[string][]byte{
			"url":    []byte(atlasIntegration.GetUrl()),
			"secret": []byte(atlasIntegration.GetSecret()),
		}))
		resource.Spec.Webhook = &akov2.WebhookIntegration{
			URLSecretRef: akoapi.LocalObjectReference{Name: secretName},
		}
	}
	if request.IndependentResource {
		resource.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ExternalProjectRef: &akov2.ExternalProjectReference{
				ID: request.ProjectID,
			},
			ConnectionSecret: &akoapi.LocalObjectReference{
				Name: resources.NormalizeAtlasName(request.Credentials, request.Dictionary),
			},
		}
	} else {
		resource.Spec.ProjectDualReference = akov2.ProjectDualReference{
			ProjectRef: &akov2common.ResourceRefNamespaced{
				Name:      request.ProjectName,
				Namespace: request.TargetNamespace,
			},
		}
	}
	return results
}

func integrationResourceName(request ThirdPartyIntegrationRequest, atlasIntegration *admin.ThirdPartyIntegration) string {
	baseName := fmt.Sprintf("%s-%s-integration", request.ProjectName, dashit(atlasIntegration.GetType()))
	return resources.NormalizeAtlasName(baseName, request.Dictionary)
}

func integrationResourceSecret(request ThirdPartyIntegrationRequest, atlasIntegration *admin.ThirdPartyIntegration) string {
	baseName := fmt.Sprintf("%s-%s-integration-secret", request.ProjectName, dashit(atlasIntegration.GetType()))
	return resources.NormalizeAtlasName(baseName, request.Dictionary)
}

func convertEnabledString(on bool) *string {
	if on {
		return pointer.Get("enabled")
	}
	return pointer.Get("disabled")
}

func newIntegrationSecret(request ThirdPartyIntegrationRequest, secretName, fieldName, value string) *corev1.Secret {
	return newIntegrationRawSecret(request, secretName, map[string][]byte{fieldName: []byte(value)})
}

func newIntegrationRawSecret(request ThirdPartyIntegrationRequest, secretName string, data map[string][]byte) *corev1.Secret {
	return secrets.NewAtlasSecretBuilder(secretName, request.TargetNamespace, request.Dictionary).
		WithData(data).WithProjectLabels(request.ProjectID, request.ProjectName).Build()
}

func dashit(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}
