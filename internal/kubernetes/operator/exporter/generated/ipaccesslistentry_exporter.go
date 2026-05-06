// Copyright 2025 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// nolint:dupl
package exporter

import (
	"context"
	"errors"
	"fmt"

	akov2generated "github.com/mongodb/mongodb-atlas-kubernetes/v2/generated/v1"
	crapi "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/crapi"
	admin "go.mongodb.org/atlas-sdk/v20250312018/admin"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

type IPAccessListEntryExporter struct {
	identifiers []string

	client     *admin.APIClient
	translator crapi.Translator
}

func (e *IPAccessListEntryExporter) Export(ctx context.Context, referencedObjects []client.Object) ([]client.Object, error) {
	var atlasResources []any
	for pageNum := 1; ; pageNum++ {
		resp, _, err := e.client.ProjectIPAccessListApi.ListAccessListEntries(ctx, e.identifiers[0]).PageNum(pageNum).Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list IPAccessListEntrys from Atlas: %w", err)
		}
		if resp == nil {
			return nil, errors.New("no response")
		}
		pageResults := resp.GetResults()
		for i := range pageResults {
			atlasResources = append(atlasResources, pageResults[i])
		}
		if len(pageResults) == 0 || len(atlasResources) >= resp.GetTotalCount() {
			break
		}
	}

	resources := make([]client.Object, 0, len(atlasResources))
	for _, atlasResource := range atlasResources {
		resource := &akov2generated.IPAccessListEntry{}
		translatedResources, err := e.translator.FromAPI(resource, atlasResource, referencedObjects...)
		if err != nil {
			return nil, fmt.Errorf("failed to translate IPAccessListEntry: %w", err)
		}

		// Edge case: Atlas returns both IP address and CIDR block when the config is an IP address.
		if resource.Spec.V20250312 != nil && resource.Spec.V20250312.Entry != nil && resource.Spec.V20250312.Entry.IpAddress != nil {
			resource.Spec.V20250312.Entry.CidrBlock = nil
		}

		resource.GetObjectKind().SetGroupVersionKind(akov2generated.GroupVersion.WithKind("IPAccessListEntry"))
		var id string
		switch {
		case resource.Spec.V20250312 != nil && resource.Spec.V20250312.Entry != nil && resource.Spec.V20250312.Entry.IpAddress != nil:
			id = *resource.Spec.V20250312.Entry.IpAddress
		case resource.Spec.V20250312 != nil && resource.Spec.V20250312.Entry != nil && resource.Spec.V20250312.Entry.CidrBlock != nil:
			id = *resource.Spec.V20250312.Entry.CidrBlock
		case resource.Spec.V20250312 != nil && resource.Spec.V20250312.Entry != nil && resource.Spec.V20250312.Entry.AwsSecurityGroup != nil:
			id = *resource.Spec.V20250312.Entry.AwsSecurityGroup
		}
		resource.SetAnnotations(map[string]string{"mongodb.com/external-id": id})

		resources = append(resources, resource)
		resources = append(resources, translatedResources...)
	}

	return resources, nil
}

func NewIPAccessListEntryExporter(client *admin.APIClient, translator crapi.Translator, identifiers []string) Exporter {
	return &IPAccessListEntryExporter{
		client:      client,
		identifiers: identifiers,
		translator:  translator,
	}
}
