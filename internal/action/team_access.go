package action

import (
	"context"

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
			RunTasks:         ta.Permissions.RunTasks,
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
	RunTasks         bool   `yaml:"run_tasks"`
}

// findTeamByID takes a list of teams and returns a matching team to the passed ID
func findTeamByID(teams []*tfe.Team, teamID string) *tfe.Team {
	for _, t := range teams {
		if t.ID == teamID {
			return t
		}
	}

	return nil
}

// ToTeamAccess takes a tfe.TeamAccess object and returns a TeamAccess object
func ToTeamAccessItems(access []*tfe.TeamAccess, teams []*tfe.Team, workspace *Workspace) (ta []TeamAccessItem, err error) {
	for _, a := range access {
		t := findTeamByID(teams, a.Team.ID)

		item := TeamAccessItem{
			Workspace: workspace,
			Access:    string(a.Access),
			TeamName:  t.Name,
		}

		if a.Team.Permissions != nil {
			item.Permissions = &TeamAccessPermissionsInput{
				Runs:             string(a.Runs),
				Variables:        string(a.Variables),
				StateVersions:    string(a.StateVersions),
				SentinelMocks:    string(a.SentinelMocks),
				WorkspaceLocking: a.WorkspaceLocking,
			}
		}

		ta = append(ta, item)
	}

	return ta, nil
}

// FetchRelatedTeamAccess finds all team access resources related to the passed workspace
func FetchRelatedTeams(ctx context.Context, client *tfe.Client, workspace *Workspace, organization string) ([]*tfe.Team, error) {
	teams, err := client.Teams.List(ctx, organization, tfe.TeamListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: maxPageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	return teams.Items, nil
}

// FetchRelatedTeamAccess finds all team access resources related to the passed workspace
func FetchRelatedTeamAccess(ctx context.Context, client *tfe.Client, workspace *Workspace) ([]*tfe.TeamAccess, error) {
	teamAccess, err := client.TeamAccess.List(ctx, tfe.TeamAccessListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: maxPageSize,
		},
		WorkspaceID: workspace.ID,
	})
	if err != nil {
		return nil, err
	}

	return teamAccess.Items, nil
}
