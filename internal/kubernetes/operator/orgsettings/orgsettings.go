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

package orgsettings

import (
	"fmt"

	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/kubernetes/operator/secrets"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store"
	akoapi "github.com/mongodb/mongodb-atlas-kubernetes/v2/api"
	akov2 "github.com/mongodb/mongodb-atlas-kubernetes/v2/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildAtlasOrgSettings(orgID string, provider store.OrgSettingsStore, creds store.CredentialsGetter, targetNs string, includeSecretData bool, dict map[string]string) (*akov2.AtlasOrgSettings, *corev1.Secret, error) {
	res, err := provider.GetOrgSettings(orgID)
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving orgsettings: %s", err)
	}

	if res == nil {
		return nil, nil, nil
	}

	secret := secrets.NewAtlasSecretBuilder(fmt.Sprintf("orgsettings-%s", orgID), targetNs, dict).
		WithData(map[string][]byte{
			secrets.CredOrgID:         []byte(""),
			secrets.CredPublicAPIKey:  []byte(""),
			secrets.CredPrivateAPIKey: []byte(""),
		}).
		Build()
	if includeSecretData {
		secret.Data = map[string][]byte{
			secrets.CredOrgID:         []byte(orgID),
			secrets.CredPublicAPIKey:  []byte(creds.PublicAPIKey()),
			secrets.CredPrivateAPIKey: []byte(creds.PrivateAPIKey()),
		}
	}

	return &akov2.AtlasOrgSettings{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AtlasOrgSettings",
			APIVersion: "atlas.mongodb.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("orgsettings-%s", orgID),
			Namespace: targetNs,
		},
		Spec: akov2.AtlasOrgSettingsSpec{
			OrgID:                                  orgID,
			ConnectionSecretRef:                    &akoapi.LocalObjectReference{Name: secret.GetName()},
			ApiAccessListRequired:                  res.ApiAccessListRequired,
			GenAIFeaturesEnabled:                   res.GenAIFeaturesEnabled,
			MaxServiceAccountSecretValidityInHours: res.MaxServiceAccountSecretValidityInHours,
			MultiFactorAuthRequired:                res.MultiFactorAuthRequired,
			RestrictEmployeeAccess:                 res.RestrictEmployeeAccess,
			SecurityContact:                        res.SecurityContact,
			StreamsCrossGroupEnabled:               res.StreamsCrossGroupEnabled,
		},
	}, secret, nil
}
