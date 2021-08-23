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
}

type TeamAccess []TeamAccessItem

type TeamAccessItem struct {
	Access      string
	Permissions *TeamAccessPermissionsInput
	TeamName    string
	TeamID      string

	Workspace *Workspace
}

// NewTeamAccess takes a team inputs and workspaces and returns a TeamAccessItem per input, per workspace
func NewTeamAccess(inputs TeamAccessInput, workspaces []*Workspace) TeamAccess {
	access := make(TeamAccess, len(inputs)*len(workspaces))

	i := 0

	for _, team := range inputs {
		for _, ws := range workspaces {
			access[i] = TeamAccessItem{
				Access:      team.Access,
				Permissions: team.Permissions,
				TeamName:    team.TeamName,
				TeamID:      team.TeamID,
				Workspace:   ws,
			}
			i = i + 1
		}
	}

	return access
}

// ToResource converts the TeamAccessItem to a Terraform resource
func (ta TeamAccessItem) ToResource() *tfeprovider.TeamAccess {
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
func (tai TeamAccessInputItem) Validate() error {
	if tai.TeamID != "" && tai.TeamName != "" {
		return fmt.Errorf("team name and team ID cannot both be set: %s, %s", tai.TeamID, tai.TeamName)
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
