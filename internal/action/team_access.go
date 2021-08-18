package action

import (
	"fmt"

	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

// TeamAccessInput is a list of team access settings
type TeamAccessInput []TeamAccessInputItem

// TeamAccessInputItem represents a single team access setting
type TeamAccessInputItem struct {
	Access      string                      `yaml:"access,omitempty"`
	Permissions *TeamAccessPermissionsInput `yaml:"permissions,omitempty"`
	TeamName    string                      `yaml:"name"`
	TeamID      string                      `yaml:"id"`

	// TODO: remove this, objects should not store data on behalf of callers
	WorkspaceName string
}

// ToResource converts the TeamAccess input to a Terraform resource
func (ta TeamAccessInputItem) ToResource() *tfeprovider.TeamAccess {
	resource := &tfeprovider.TeamAccess{
		Access: ta.Access,
		TeamID: ta.TeamID,
	}

	if ta.Permissions != nil {
		resource.Permissions = &tfeprovider.TeamAccessPermissions{
			Runs:             ta.Permissions.Runs,
			Variables:        ta.Permissions.Variables,
			StateVersions:    ta.Permissions.StateVersions,
			SentinelMocks:    ta.Permissions.SentinelMocks,
			WorkspaceLocking: ta.Permissions.WorkspaceLocking,
		}
	}

	return resource
}

// Validate validates the content of the team access input
func (ta TeamAccessInputItem) Validate() error {
	if ta.TeamID != "" && ta.TeamName != "" {
		return fmt.Errorf("team name and team ID cannot both be set: %s, %s", ta.TeamID, ta.TeamName)
	}

	return nil
}

type TeamAccessPermissionsInput struct {
	Runs             string `yaml:"runs"`
	Variables        string `yaml:"variables"`
	StateVersions    string `yaml:"state_versions"`
	SentinelMocks    string `yaml:"sentinel_mocks"`
	WorkspaceLocking bool   `yaml:"workspace_locking"`
}
