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

	"github.com/google/uuid"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/features"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/resources"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	akoapi "github.com/mongodb/mongodb-atlas-kubernetes/v2/api"
	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1"
	akov2common "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/common"
	akov2provider "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/provider"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1/status"
	"go.mongodb.org/atlas-sdk/v20241113004/admin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkContainersRequest struct {
	ProjectName         string
	ProjectID           string
	TargetNamespace     string
	Version             string
	Credentials         string
	IndependentResource bool
	Dictionary          map[string]string
}

func BuildNetworkContainers(
	provider store.NetworkContainerLister,
	request NetworkContainersRequest,
) ([]akov2.AtlasNetworkContainer, error) {
	atlasContainers, err := provider.NetworkContainers(request.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list network container from Atlas: %w", err)
	}

	kubeContainers := make([]akov2.AtlasNetworkContainer, 0, len(atlasContainers))
	for _, atlasContainer := range atlasContainers {
		kubeContainers = append(kubeContainers, convertContainerToKubernetes(request, &atlasContainer))
	}
	return kubeContainers, nil
}

func convertContainerToKubernetes(request NetworkContainersRequest, atlasContainer *admin.CloudProviderContainer) akov2.AtlasNetworkContainer {
	resource := akov2.AtlasNetworkContainer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasNetworkContainer",
			APIVersion: fmt.Sprintf("%s/%s", akov2.GroupVersion.Group, akov2.GroupVersion.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.NormalizeAtlasName(request.ProjectName+"-container-"+randomSuffix(5), request.Dictionary),
			Namespace: request.TargetNamespace,
			Labels: map[string]string{
				features.ResourceVersion: request.Version,
			},
		},
		Spec: akov2.AtlasNetworkContainerSpec{
			Provider: atlasContainer.GetProviderName(),
			AtlasNetworkContainerConfig: akov2.AtlasNetworkContainerConfig{
				ID:        atlasContainer.GetId(),
				CIDRBlock: atlasContainer.GetAtlasCidrBlock(),
			},
		},
		Status: status.AtlasNetworkContainerStatus{
			Common: akoapi.Common{
				Conditions: []akoapi.Condition{},
			},
		},
	}
	switch atlasContainer.GetProviderName() {
	case string(akov2provider.ProviderAWS):
		resource.Spec.Region = atlasContainer.GetRegionName()
	case string(akov2provider.ProviderAzure):
		resource.Spec.Region = atlasContainer.GetRegion()
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

func randomSuffix(size int) string {
	return uuid.NewString()[:size]
}
