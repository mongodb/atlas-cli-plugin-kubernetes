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
	akov2provider "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/provider"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/atlas-sdk/v20241113004/admin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPeerings(t *testing.T) {
	projectID := "project-np-id"
	projectName := "projectName-np"
	targetNamespace := "npNamespace"
	credentialName := "np-creds"
	version := "2.8.0"

	for _, tc := range []struct {
		title               string
		atlasPeerings       []admin.BaseNetworkPeeringConnectionSettings
		independentResource bool
		wantResources       []akov2.AtlasNetworkPeering
	}{
		{
			title:         "no peerings",
			atlasPeerings: []admin.BaseNetworkPeeringConnectionSettings{},
			wantResources: []akov2.AtlasNetworkPeering{},
		},
		{
			title: "AWS peering properly exported",
			atlasPeerings: []admin.BaseNetworkPeeringConnectionSettings{
				{
					Id:                  pointer.Get("peering-id"),
					ContainerId:         "container-id",
					ProviderName:        pointer.Get(string(akov2provider.ProviderAWS)),
					AccepterRegionName:  pointer.Get("US_EAST_1"),
					AwsAccountId:        pointer.Get("some-aws-account"),
					ConnectionId:        pointer.Get("connection id"),
					RouteTableCidrBlock: pointer.Get("10.0.0.0/19"),
					VpcId:               pointer.Get("some-vpc"),
				},
			},
			wantResources: []akov2.AtlasNetworkPeering{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkPeering",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-peering-",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkPeeringSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						ContainerRef: akov2.ContainerDualReference{
							ID: "container-id",
						},
						AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
							ID:       "peering-id",
							Provider: string(akov2provider.ProviderAWS),
							AWSConfiguration: &akov2.AWSNetworkPeeringConfiguration{
								AccepterRegionName:  "US_EAST_1",
								AWSAccountID:        "some-aws-account",
								RouteTableCIDRBlock: "10.0.0.0/19",
								VpcID:               "some-vpc",
							},
						},
					},
					Status: status.AtlasNetworkPeeringStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
			},
		},
		{
			title: "Azure and GCP peerings properly exported",
			atlasPeerings: []admin.BaseNetworkPeeringConnectionSettings{
				{
					ContainerId: "container-id-0",
					Id: pointer.Get("peering-id-0"),
					ProviderName: pointer.Get(string(akov2provider.ProviderAzure)),
					AzureDirectoryId:    pointer.Get("azure-directory-id"),
					AzureSubscriptionId: pointer.Get("azure-subscription-id"),
					ResourceGroupName:   pointer.Get("resource-group-name"),
					VnetName:            pointer.Get("vnet-name"),
				},
				{
					ContainerId: "container-id-1",
					Id: pointer.Get("peering-id-1"),
					ProviderName: pointer.Get(string(akov2provider.ProviderGCP)),
					GcpProjectId:        pointer.Get("gcp-project-id"),
					NetworkName:         pointer.Get("network-name"),
				},
			},
			wantResources: []akov2.AtlasNetworkPeering{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkPeering",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-peering-",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkPeeringSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						ContainerRef: akov2.ContainerDualReference{
							ID: "container-id-0",
						},
						AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
							ID:       "peering-id-0",
							Provider: string(akov2provider.ProviderAzure),
							AzureConfiguration: &akov2.AzureNetworkPeeringConfiguration{
								AzureDirectoryID:    "azure-directory-id",
								AzureSubscriptionID: "azure-subscription-id",
								ResourceGroupName:   "resource-group-name",
								VNetName:            "vnet-name",
							},
						},
					},
					Status: status.AtlasNetworkPeeringStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkPeering",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-peering-",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkPeeringSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						ContainerRef: akov2.ContainerDualReference{
							ID: "container-id-1",
						},
						AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
							ID:       "peering-id-1",
							Provider: string(akov2provider.ProviderGCP),
							GCPConfiguration: &akov2.GCPNetworkPeeringConfiguration{
								GCPProjectID: "gcp-project-id",
								NetworkName:  "network-name",
							},
						},
					},
					Status: status.AtlasNetworkPeeringStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
			},
		},
		{
			title: "AWS, Azure and GCP peering properly exported with IDs",
			independentResource: true,
			atlasPeerings: []admin.BaseNetworkPeeringConnectionSettings{
				{
					Id:                  pointer.Get("peering-id"),
					ContainerId:         "container-id",
					ProviderName:        pointer.Get(string(akov2provider.ProviderAWS)),
					AccepterRegionName:  pointer.Get("US_EAST_1"),
					AwsAccountId:        pointer.Get("some-aws-account"),
					ConnectionId:        pointer.Get("connection id"),
					RouteTableCidrBlock: pointer.Get("10.0.0.0/19"),
					VpcId:               pointer.Get("some-vpc"),
				},
				{
					ContainerId: "container-id-0",
					Id: pointer.Get("peering-id-0"),
					ProviderName: pointer.Get(string(akov2provider.ProviderAzure)),
					AzureDirectoryId:    pointer.Get("azure-directory-id"),
					AzureSubscriptionId: pointer.Get("azure-subscription-id"),
					ResourceGroupName:   pointer.Get("resource-group-name"),
					VnetName:            pointer.Get("vnet-name"),
				},
				{
					ContainerId: "container-id-1",
					Id: pointer.Get("peering-id-1"),
					ProviderName: pointer.Get(string(akov2provider.ProviderGCP)),
					GcpProjectId:        pointer.Get("gcp-project-id"),
					NetworkName:         pointer.Get("network-name"),
				},
			},
			wantResources: []akov2.AtlasNetworkPeering{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkPeering",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-peering-",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkPeeringSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						ContainerRef: akov2.ContainerDualReference{
							ID: "container-id",
						},
						AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
							ID:       "peering-id",
							Provider: string(akov2provider.ProviderAWS),
							AWSConfiguration: &akov2.AWSNetworkPeeringConfiguration{
								AccepterRegionName:  "US_EAST_1",
								AWSAccountID:        "some-aws-account",
								RouteTableCIDRBlock: "10.0.0.0/19",
								VpcID:               "some-vpc",
							},
						},
					},
					Status: status.AtlasNetworkPeeringStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkPeering",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-peering-",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkPeeringSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						ContainerRef: akov2.ContainerDualReference{
							ID: "container-id-0",
						},
						AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
							ID:       "peering-id-0",
							Provider: string(akov2provider.ProviderAzure),
							AzureConfiguration: &akov2.AzureNetworkPeeringConfiguration{
								AzureDirectoryID:    "azure-directory-id",
								AzureSubscriptionID: "azure-subscription-id",
								ResourceGroupName:   "resource-group-name",
								VNetName:            "vnet-name",
							},
						},
					},
					Status: status.AtlasNetworkPeeringStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkPeering",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-peering-",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkPeeringSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						ContainerRef: akov2.ContainerDualReference{
							ID: "container-id-1",
						},
						AtlasNetworkPeeringConfig: akov2.AtlasNetworkPeeringConfig{
							ID:       "peering-id-1",
							Provider: string(akov2provider.ProviderGCP),
							GCPConfiguration: &akov2.GCPNetworkPeeringConfiguration{
								GCPProjectID: "gcp-project-id",
								NetworkName:  "network-name",
							},
						},
					},
					Status: status.AtlasNetworkPeeringStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			ctl := gomock.NewController(t)
			npStore := mocks.NewMockPeeringConnectionLister(ctl)
			dictionary := resources.AtlasNameToKubernetesName()

			npStore.EXPECT().PeeringConnections(projectID).Return(tc.atlasPeerings, nil)

			peerings, err := BuildNetworkPeerings(
				npStore,
				NetworkPeeringsRequest{
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
			assert.Equal(t, tc.wantResources, deRandomizePeerings(peerings))
		})
	}
}

func deRandomizePeerings(peerings []akov2.AtlasNetworkPeering) []akov2.AtlasNetworkPeering {
	predictablePeerings := make([]akov2.AtlasNetworkPeering, 0, len(peerings))
	for _, peering := range peerings {
		predictablePeering := peering.DeepCopy()
		predictablePeering.Name = predictablePeering.Name[:len(predictablePeering.Name)-5]
		predictablePeerings = append(predictablePeerings, *predictablePeering)
	}
	return predictablePeerings
}
