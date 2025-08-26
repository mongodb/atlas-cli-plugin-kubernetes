package store

import (
	"fmt"

	atlasv2 "go.mongodb.org/atlas-sdk/v20241113004/admin"
)

//go:generate mockgen -destination=../mocks/mock_orgsettings.go -package=mocks github.com/mongodb/atlas-cli-plugin-kubernetes/internal/store OrgSettingsDescriber,OrgSettingsUpdater

type OrgSettingsDescriber interface {
	GetOrgSettings(string) (*atlasv2.OrganizationSettings, error)
}

type OrgSettingsUpdater interface {
	UpdateOrgSettings(string, *atlasv2.OrganizationSettings) (*atlasv2.OrganizationSettings, error)
}

func (s *Store) GetOrgSettings(orgID string) (*atlasv2.OrganizationSettings, error) {
	resp, _, err := s.clientv2.OrganizationsApi.GetOrganizationSettings(s.ctx, orgID).Execute()
	return resp, err
}

func (s *Store) UpdateOrgSettings(orgID string, settings *atlasv2.OrganizationSettings) (*atlasv2.OrganizationSettings, error) {
	resp, httpResp, err := s.clientv2.OrganizationsApi.UpdateOrganizationSettings(s.ctx, orgID, settings).Execute()
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != 200 && httpResp.StatusCode != 201 {
		return nil, fmt.Errorf("error updating organization settings: %s", httpResp.Status)
	}
	return resp, nil
}
