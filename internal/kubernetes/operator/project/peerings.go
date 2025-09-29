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
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	akoapi "github.com/mongodb/mongodb-atlas-kubernetes/v2/api"
	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1"
	akov2common "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/common"
	akov2provider "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/provider"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/status"
	"go.mongodb.org/atlas-sdk/v20250312006/admin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkPeeringsRequest struct {
	ProjectName         string
	ProjectID           string
	TargetNamespace     string
	Version             string
	Credentials         string
	IndependentResource bool
	Dictionary          map[string]string
}

func BuildNetworkPeerings(
	provider store.PeeringConnectionLister,
	request NetworkPeeringsRequest,
) ([]akov2.AtlasNetworkPeering, error) {
	atlasPeerings, err := provider.PeeringConnections(request.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list peering connections from Atlas: %w", err)
	}

	kubePeerings := make([]akov2.AtlasNetworkPeering, 0, len(atlasPeerings))
	for _, atlasPeering := range atlasPeerings {
		kubePeerings = append(kubePeerings, convertPeeringToKubernetes(request, &atlasPeering))
	}
	return kubePeerings, nil
}

func convertPeeringToKubernetes(request NetworkPeeringsRequest, atlasPeering *admin.BaseNetworkPeeringConnectionSettings) akov2.AtlasNetworkPeering {
	resource := akov2.AtlasNetworkPeering{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasNetworkPeering",
			APIVersion: fmt.Sprintf("%s/%s", akov2.GroupVersion.Group, akov2.GroupVersion.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      peeringResourceName(request, atlasPeering),
			Namespace: request.TargetNamespace,
			Labels: map[string]string{
				features.ResourceVersion: request.Version,
			},
		},
		Spec: akov2.AtlasNetworkPeeringSpec{
			ContainerRef: akov2.ContainerDualReference{
				ID: atlasPeering.GetContainerId(),
			},
			AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
				ID:       atlasPeering.GetId(),
				Provider: atlasPeering.GetProviderName(),
			},
		},
		Status: status.AtlasNetworkPeeringStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}
	switch atlasPeering.GetProviderName() {
	case string(akov2provider.ProviderAWS):
		resource.Spec.AWSConfiguration = &akov2.AWSNetworkPeeringConfiguration{
			AccepterRegionName:  atlasPeering.GetAccepterRegionName(),
			AWSAccountID:        atlasPeering.GetAwsAccountId(),
			RouteTableCIDRBlock: atlasPeering.GetRouteTableCidrBlock(),
			VpcID:               atlasPeering.GetVpcId(),
		}
	case string(akov2provider.ProviderAzure):
		resource.Spec.AzureConfiguration = &akov2.AzureNetworkPeeringConfiguration{
			AzureDirectoryID:    atlasPeering.GetAzureDirectoryId(),
			AzureSubscriptionID: atlasPeering.GetAzureSubscriptionId(),
			ResourceGroupName:   atlasPeering.GetResourceGroupName(),
			VNetName:            atlasPeering.GetVnetName(),
		}
	case string(akov2provider.ProviderGCP):
		resource.Spec.GCPConfiguration = &akov2.GCPNetworkPeeringConfiguration{
			GCPProjectID: atlasPeering.GetGcpProjectId(),
			NetworkName:  atlasPeering.GetNetworkName(),
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
	return resource
}

func peeringResourceName(request NetworkPeeringsRequest, atlasPeering *admin.BaseNetworkPeeringConnectionSettings) string {
	provider := atlasPeering.GetProviderName()
	conn := ""
	switch provider {
	case string(akov2provider.ProviderAWS):
		region := strings.ReplaceAll(atlasPeering.GetAccepterRegionName(), "_", "")
		conn = strings.ToLower(fmt.Sprintf("%s-%s", region, atlasPeering.GetVpcId()))
	case string(akov2provider.ProviderAzure):
		conn = strings.ToLower(fmt.Sprintf("%s-%s", atlasPeering.GetAzureSubscriptionId(), atlasPeering.GetVnetName()))
	case string(akov2provider.ProviderGCP):
		conn = strings.ToLower(fmt.Sprintf("%s-%s", atlasPeering.GetGcpProjectId(), atlasPeering.GetNetworkName()))
	}
	baseName := fmt.Sprintf("%s-peering-%s-%s", request.ProjectName, provider, conn)
	return resources.NormalizeAtlasName(baseName, request.Dictionary)
}
