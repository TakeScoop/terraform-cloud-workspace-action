package main

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type TeamAccess struct {
	Access        string                 `yaml:"access,omitempty"`
	Permissions   *TeamAccessPermissions `yaml:"permissions,omitempty"`
	TeamName      string                 `yaml:"team_name"`
	WorkspaceName string
	ResourceName  string
}

type TeamAccessPermissions struct {
	Runs             string `yaml:"runs" json:"runs"`
	Variables        string `yaml:"variables" json:"variables"`
	StateVersions    string `yaml:"state_versions" json:"state_versions"`
	SentinelMocks    string `yaml:"sentinel_mocks" json:"sentinel_mocks"`
	WorkspaceLocking bool   `yaml:"workspace_locking" json:"workspace_locking"`
}

// MergeWorkspaceIDs returns a new slice of TeamAccess structs
func MergeWorkspaceIDs(teamAccess map[string]TeamAccess, workspaces []*Workspace) []TeamAccess {
	ts := make([]TeamAccess, len(teamAccess)*len(workspaces))

	i := 0
	for resourceName, team := range teamAccess {
		for _, ws := range workspaces {
			team.WorkspaceName = ws.Name
			team.ResourceName = resourceName
			ts[i] = team
			i = i + 1
		}
	}

	return ts
}

// Import imports a team access resource by looking up an existing relation
func (ta *TeamAccess) Import(ctx context.Context, tf *tfexec.Terraform, client *tfe.Client, organization string, opts ...tfexec.ImportOption) error {
	if strings.HasPrefix("${", ta.TeamName) {
		return fmt.Errorf("Team access only supports importing statically named teams, but an expression (${}) was found. Convert the team name to a static string to allow import.")
	}

	address := fmt.Sprintf("tfe_team_access[\"%s-%s\"]", ta.WorkspaceName, ta.ResourceName)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		fmt.Printf("Team access %q already exists in state, skipping import\n", address)
		return nil
	}

	fmt.Printf("Importing team access: %q\n", address)

	ws, err := client.Workspaces.Read(ctx, organization, ta.WorkspaceName)
	if err != nil {
		return err
	}

	teamAccess, err := client.TeamAccess.List(ctx, tfe.TeamAccessListOptions{
		WorkspaceID: &ws.ID,
	})
	if err != nil {
		return err
	}

	var teamAccessID string

	for _, access := range teamAccess.Items {
		if access.Team.Name == ta.TeamName {
			teamAccessID = access.ID
		}
	}

	if teamAccessID == "" {
		fmt.Printf("Team access %q for workspace %q not found, skipping import\n", ta.TeamName, ta.WorkspaceName)
		return nil
	}

	importID := fmt.Sprintf("%s/%s/%s", organization, ta.WorkspaceName, teamAccessID)

	if err = tf.Import(ctx, address, importID, opts...); err != nil {
		return err
	}

	fmt.Printf("Team access %q successfully imported\n", importID)

	return nil
}
