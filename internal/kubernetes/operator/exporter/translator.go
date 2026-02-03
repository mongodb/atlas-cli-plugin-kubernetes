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

package exporter

import (
	"fmt"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/crapi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewTranslator creates a new Crapi translator for the given CRD.
// It extracts the CRD version from the CRD spec and uses the provided SDK version
// to ensure compatibility with the SDK client.
func NewTranslator(scheme *runtime.Scheme, crd *apiextensionsv1.CustomResourceDefinition, sdkVersion string) (crapi.Translator, error) {
	if crd == nil {
		return nil, fmt.Errorf("CRD is nil")
	}

	crdVersion, err := extractCRDVersion(crd)
	if err != nil {
		return nil, fmt.Errorf("failed to extract CRD version: %w", err)
	}

	translator, err := crapi.NewTranslator(scheme, crd, crdVersion, sdkVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to create translator for CRD %s: %w", crd.Name, err)
	}

	return translator, nil
}

// extractCRDVersion extracts the CRD version from the CRD spec.
func extractCRDVersion(crd *apiextensionsv1.CustomResourceDefinition) (string, error) {
	if len(crd.Spec.Versions) == 0 {
		return "", fmt.Errorf("CRD has no versions defined")
	}
	return crd.Spec.Versions[0].Name, nil
}
