package action

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

// TeamAccessInput is a list of team access settings
type TeamAccessInput []TeamAccessInputItem

// TeamAccessInputItem represents a single team access setting
type TeamAccessInputItem struct {
	Access      string                      `yaml:"access,omitempty"`
	Permissions *TeamAccessPermissionsInput `yaml:"permissions,omitempty"`
	TeamName    string                      `yaml:"name"`
}

type TeamAccess []TeamAccessItem

type TeamAccessItem struct {
	Access      string
	Permissions *TeamAccessPermissionsInput
	TeamName    string

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

type TeamAccessPermissionsInput struct {
	Runs             string `yaml:"runs"`
	Variables        string `yaml:"variables"`
	StateVersions    string `yaml:"state_versions"`
	SentinelMocks    string `yaml:"sentinel_mocks"`
	WorkspaceLocking bool   `yaml:"workspace_locking"`
}

// FindRelatedTeamAccess returns a list of workspace related team access resources
func FindRelatedTeamAccess(ctx context.Context, client *tfe.Client, workspace *Workspace, organization string) (TeamAccess, error) {
	if workspace.ID == nil {
		fmt.Println("no WS ID found")
		return TeamAccess{}, nil
	}

	fmt.Printf("ws ID: %s\n", *workspace.ID)

	tas, err := client.TeamAccess.List(ctx, tfe.TeamAccessListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
		WorkspaceID: workspace.ID,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println("related team access", tas)

	var access TeamAccess

	for _, ta := range tas.Items {
		item := TeamAccessItem{
			Workspace: workspace,
			Access:    string(ta.Access),
			TeamName:  ta.Team.Name,
		}

		if ta.Team.Permissions != nil {
			item.Permissions = &TeamAccessPermissionsInput{
				Runs:             string(ta.Runs),
				Variables:        string(ta.Variables),
				StateVersions:    string(ta.StateVersions),
				SentinelMocks:    string(ta.SentinelMocks),
				WorkspaceLocking: ta.WorkspaceLocking,
			}
		}

		access = append(access, item)
	}

	return access, nil
}
