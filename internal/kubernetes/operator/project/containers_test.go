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

func TestBuildContainers(t *testing.T) {
	projectID := "project-np-id"
	projectName := "projectName-np"
	targetNamespace := "npNamespace"
	credentialName := "np-creds"
	version := features.LatestOperatorMajorVersion

	for _, tc := range []struct {
		title               string
		atlasContainers     []admin.CloudProviderContainer
		independentResource bool
		wantResources       []akov2.AtlasNetworkContainer
	}{
		{
			title:           "no containers",
			atlasContainers: []admin.CloudProviderContainer{},
			wantResources:   []akov2.AtlasNetworkContainer{},
		},
		{
			title: "AWS container properly exported",
			atlasContainers: []admin.CloudProviderContainer{
				{
					Id:             pointer.Get("some-id"),
					ProviderName:   pointer.Get(string(akov2provider.ProviderAWS)),
					Provisioned:    pointer.Get(true),
					AtlasCidrBlock: pointer.Get("10.0.0.0/18"),
					RegionName:     pointer.Get("US_EAST_1"),
					VpcId:          pointer.Get("some-vpc"),
				},
			},
			wantResources: []akov2.AtlasNetworkContainer{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkContainer",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-container-aws-useast1",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkContainerSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						Provider: string(akov2provider.ProviderAWS),
						AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
							ID:        "some-id",
							Region:    "US_EAST_1",
							CIDRBlock: "10.0.0.0/18",
						},
					},
					Status: status.AtlasNetworkContainerStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
			},
		},
		{
			title: "Azure and GCP containers properly exported",
			atlasContainers: []admin.CloudProviderContainer{
				{
					Id:                  pointer.Get("some-id-azure"),
					ProviderName:        pointer.Get(string(akov2provider.ProviderAzure)),
					Provisioned:         pointer.Get(true),
					AtlasCidrBlock:      pointer.Get("11.0.0.0/18"),
					AzureSubscriptionId: pointer.Get("some-subscription-id"),
					Region:              pointer.Get("US_CENTRAL"),
					VnetName:            pointer.Get("some-vnet"),
				},
				{
					Id:             pointer.Get("some-id-gcp"),
					ProviderName:   pointer.Get(string(akov2provider.ProviderGCP)),
					Provisioned:    pointer.Get(false),
					AtlasCidrBlock: pointer.Get("12.0.0.0/18"),
					GcpProjectId:   pointer.Get("project"),
					NetworkName:    pointer.Get("network-name"),
				},
			},
			wantResources: []akov2.AtlasNetworkContainer{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkContainer",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-container-azure-uscentral",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkContainerSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						Provider: string(akov2provider.ProviderAzure),
						AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
							ID:        "some-id-azure",
							Region:    "US_CENTRAL",
							CIDRBlock: "11.0.0.0/18",
						},
					},
					Status: status.AtlasNetworkContainerStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkContainer",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-container-gcp-global",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkContainerSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ProjectRef: &common.ResourceRefNamespaced{
								Name:      projectName,
								Namespace: targetNamespace,
							},
						},
						Provider: string(akov2provider.ProviderGCP),
						AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
							ID:        "some-id-gcp",
							CIDRBlock: "12.0.0.0/18",
						},
					},
					Status: status.AtlasNetworkContainerStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
			},
		},
		{
			title: "AWS, Azure and GCP container properly exported with IDs",
			atlasContainers: []admin.CloudProviderContainer{
				{
					Id:             pointer.Get("some-id"),
					ProviderName:   pointer.Get(string(akov2provider.ProviderAWS)),
					Provisioned:    pointer.Get(true),
					AtlasCidrBlock: pointer.Get("10.0.0.0/18"),
					RegionName:     pointer.Get("US_EAST_1"),
					VpcId:          pointer.Get("some-vpc"),
				},
				{
					Id:                  pointer.Get("some-id-azure"),
					ProviderName:        pointer.Get(string(akov2provider.ProviderAzure)),
					Provisioned:         pointer.Get(true),
					AtlasCidrBlock:      pointer.Get("11.0.0.0/18"),
					AzureSubscriptionId: pointer.Get("some-subscription-id"),
					Region:              pointer.Get("US_CENTRAL"),
					VnetName:            pointer.Get("some-vnet"),
				},
				{
					Id:             pointer.Get("some-id-gcp"),
					ProviderName:   pointer.Get(string(akov2provider.ProviderGCP)),
					Provisioned:    pointer.Get(false),
					AtlasCidrBlock: pointer.Get("12.0.0.0/18"),
					GcpProjectId:   pointer.Get("gcp-project"),
					NetworkName:    pointer.Get("gcp-network"),
				},
			},
			independentResource: true,
			wantResources: []akov2.AtlasNetworkContainer{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkContainer",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-container-aws-useast1",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkContainerSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						Provider: string(akov2provider.ProviderAWS),
						AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
							ID:        "some-id",
							Region:    "US_EAST_1",
							CIDRBlock: "10.0.0.0/18",
						},
					},
					Status: status.AtlasNetworkContainerStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkContainer",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-container-azure-uscentral",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkContainerSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						Provider: string(akov2provider.ProviderAzure),
						AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
							ID:        "some-id-azure",
							Region:    "US_CENTRAL",
							CIDRBlock: "11.0.0.0/18",
						},
					},
					Status: status.AtlasNetworkContainerStatus{
						Common: api.Common{
							Conditions: []api.Condition{},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "AtlasNetworkContainer",
						APIVersion: "atlas.mongodb.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      strings.ToLower(projectName) + "-container-gcp-global",
						Namespace: targetNamespace,
						Labels: map[string]string{
							features.ResourceVersion: version,
						},
					},
					Spec: akov2.AtlasNetworkContainerSpec{
						ProjectDualReference: akov2.ProjectDualReference{
							ExternalProjectRef: &akov2.ExternalProjectReference{
								ID: projectID,
							},
							ConnectionSecret: &api.LocalObjectReference{
								Name: credentialName,
							},
						},
						Provider: string(akov2provider.ProviderGCP),
						AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
							ID:        "some-id-gcp",
							CIDRBlock: "12.0.0.0/18",
						},
					},
					Status: status.AtlasNetworkContainerStatus{
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
			ncStore := mocks.NewMockNetworkContainerLister(ctl)
			dictionary := resources.AtlasNameToKubernetesName()

			ncStore.EXPECT().NetworkContainers(projectID).Return(tc.atlasContainers, nil)

			containers, err := BuildNetworkContainers(
				ncStore,
				NetworkContainersRequest{
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
			assert.Equal(t, tc.wantResources, containers)
		})
	}
}
