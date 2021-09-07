package action

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/sethvargo/go-githubactions"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfconfig"
)

var maxPageSize int = 100

func shouldImport(ctx context.Context, tf TerraformCLI, address string) (bool, error) {
	state, err := tf.Show(ctx)
	if err != nil {
		return false, err
	}

	if state.Values == nil {
		return true, nil
	}

	for _, r := range state.Values.RootModule.Resources {
		if address == r.Address {
			return false, nil
		}
	}

	return true, nil
}

type TerraformCLI interface {
	Show(context.Context, ...tfexec.ShowOption) (*tfjson.State, error)
	Import(context.Context, string, string, ...tfexec.ImportOption) error
}

// ImportWorkspace imports the passed workspace into Terraform state
func ImportWorkspace(ctx context.Context, tf TerraformCLI, client *tfe.Client, workspace *Workspace, organization string, opts ...tfexec.ImportOption) error {
	if workspace.ID == nil {
		githubactions.Infof("Workspace %q not found, skipping import\n", workspace.Name)
		return nil
	}

	address := fmt.Sprintf("tfe_workspace.workspace[%q]", workspace.Name)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		githubactions.Infof("Workspace %q already exists in state, skipping import\n", workspace.Name)
		return nil
	}

	githubactions.Infof("Importing workspace: %s\n", workspace.Name)

	err = tf.Import(ctx, address, *workspace.ID, opts...)
	if err != nil {
		return err
	}

	githubactions.Infof("Successful workspace import: %s\n", workspace.Name)

	return nil
}

func fetchVariableByKey(ctx context.Context, client *tfe.Client, key string, workspaceID string, page int) (*tfe.Variable, error) {
	vs, err := client.Variables.List(ctx, workspaceID, tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: maxPageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, v := range vs.Items {
		if v.Key == key {
			return v, nil
		}
	}

	if vs.NextPage > page {
		return fetchVariableByKey(ctx, client, key, workspaceID, vs.NextPage)
	}

	return nil, nil
}

// ImportVariable imports the passed variable into Terraform state
func ImportVariable(ctx context.Context, tf TerraformCLI, client *tfe.Client, key string, workspace *Workspace, organization string, opts ...tfexec.ImportOption) error {
	if workspace.ID == nil {
		githubactions.Infof("Workspace %q not found, skipping import\n", workspace.Name)
		return nil
	}

	address := fmt.Sprintf("tfe_variable.%s-%s", workspace.Name, key)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		githubactions.Infof("Variable %q already exists in state, skipping import\n", address)
		return nil
	}

	githubactions.Infof("Importing variable: %q\n", address)

	v, err := fetchVariableByKey(ctx, client, key, *workspace.ID, 1)
	if err != nil {
		return err
	}

	if v == nil {
		githubactions.Infof("Variable %q for workspace %q not found, skipping import\n", key, workspace.Name)
		return nil
	}

	importID := fmt.Sprintf("%s/%s/%s", organization, workspace.Name, v.ID)

	err = tf.Import(ctx, address, importID, opts...)
	if err != nil {
		return err
	}

	githubactions.Infof("Variable %q successfully imported\n", importID)

	return nil
}

// GetTeam returns a Team object if a team matching the passed name is found in the target Terraform account, nil is returned if the team is not found
func GetTeam(ctx context.Context, client *tfe.Client, teamName string, organization string) (*tfe.Team, error) {
	teams, err := client.Teams.List(ctx, organization, tfe.TeamListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: maxPageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, t := range teams.Items {
		if t.Name == teamName {
			return t, nil
		}
	}

	return nil, nil
}

// ImportTeamAccess imports a team access resource by looking up an existing relation
func ImportTeamAccess(ctx context.Context, tf TerraformCLI, client *tfe.Client, organization string, workspace *Workspace, teamName string, opts ...tfexec.ImportOption) error {
	if workspace.ID == nil {
		githubactions.Infof("Workspace %q not found, skipping import\n", workspace.Name)
		return nil
	}

	team, err := GetTeam(ctx, client, teamName, organization)
	if err != nil {
		return err
	}

	if team == nil {
		return fmt.Errorf("team %q not found", teamName)
	}

	address := fmt.Sprintf("tfe_team_access.teams[\"%s-%s\"]", workspace.Name, team.ID)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		githubactions.Infof("Team access %q already exists in state, skipping import\n", address)
		return nil
	}

	githubactions.Infof("Importing team access: %q\n", address)

	teamAccess, err := client.TeamAccess.List(ctx, tfe.TeamAccessListOptions{
		WorkspaceID: workspace.ID,
	})
	if err != nil {
		return err
	}

	var teamAccessID string

	for _, access := range teamAccess.Items {
		if access.Team.ID == team.ID {
			teamAccessID = access.ID
		}
	}

	if teamAccessID == "" {
		githubactions.Infof("Team access %q for workspace %q not found, skipping import\n", teamName, workspace.Name)
		return nil
	}

	importID := fmt.Sprintf("%s/%s/%s", organization, workspace.Name, teamAccessID)

	if err = tf.Import(ctx, address, importID, opts...); err != nil {
		return err
	}

	githubactions.Infof("Team access %q successfully imported\n", importID)

	return nil
}

// ImportWorkspaceResources discovers and imports resources related to the passed workspace
func ImportWorkspaceResources(ctx context.Context, client *tfe.Client, tf *tfexec.Terraform, filePath string, workspace *Workspace, organization string, providers []Provider) error {
	module := NewModule()

	wsConfig, err := NewWorkspaceResource(ctx, client, []*Workspace{workspace}, &WorkspaceResourceOptions{})
	if err != nil {
		return err
	}

	module.AppendResource("tfe_workspace", "workspace", wsConfig)

	variables, err := FindRelatedVariables(ctx, client, workspace, organization)
	if err != nil {
		return err
	}

	for _, v := range variables {
		module.AppendResource("tfe_variable", fmt.Sprintf("%s-%s", v.Workspace.Name, v.Key), v.ToResource())
	}

	teamAccess, err := FindRelatedTeamAccess(ctx, client, workspace, organization)
	if err != nil {
		return err
	}

	AppendTeamAccess(module, teamAccess, organization)

	AddProviders(module, providers)

	if err := WriteModuleFile(module, filePath); err != nil {
		return err
	}

	if err := ImportWorkspace(ctx, tf, client, workspace, organization); err != nil {
		return err
	}

	for _, v := range variables {
		err := ImportVariable(ctx, tf, client, v.Key, v.Workspace, organization)
		if err != nil {
			return err
		}
	}

	for _, access := range teamAccess {
		if err := ImportTeamAccess(ctx, tf, client, organization, access.Workspace, access.TeamName); err != nil {
			return err
		}
	}

	return nil
}

// ImportResources discovers and imports resources related to the passed workspaces
func ImportResources(ctx context.Context, client *tfe.Client, tf *tfexec.Terraform, module *tfconfig.Module, filePath string, workspaces []*Workspace, organization string, providers []Provider) error {
	for _, ws := range workspaces {
		if err := ImportWorkspaceResources(ctx, client, tf, filePath, ws, organization, providers); err != nil {
			return err
		}
	}

	if err := WriteModuleFile(module, filePath); err != nil {
		return err
	}

	return nil
}
