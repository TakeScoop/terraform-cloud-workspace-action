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

	address := fmt.Sprintf("tfe_workspace.workspace[%q]", workspace.Workspace)

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

// ImportVariable imports the passed variable into Terraform state
func ImportVariable(ctx context.Context, tf TerraformCLI, v *tfe.Variable, workspace *Workspace, organization string, opts ...tfexec.ImportOption) error {
	if workspace.ID == nil {
		githubactions.Infof("Workspace %q not found, skipping import\n", workspace.Name)
		return nil
	}

	address := fmt.Sprintf("tfe_variable.%s-%s", workspace.Workspace, v.Key)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		githubactions.Infof("Variable %q already exists in state, skipping import\n", address)
		return nil
	}

	githubactions.Infof("Importing variable: %q\n", address)

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
func ImportTeamAccess(ctx context.Context, tf TerraformCLI, access *tfe.TeamAccess, workspace *Workspace, organization string, opts ...tfexec.ImportOption) error {
	if workspace.ID == nil {
		githubactions.Infof("Workspace %q not found, skipping team access import\n", workspace.Name)
		return nil
	}

	address := fmt.Sprintf("tfe_team_access.teams[\"%s-%s\"]", workspace.Workspace, access.Team.ID)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		githubactions.Infof("Team access %q already exists in state, skipping import\n", address)
		return nil
	}

	githubactions.Infof("Importing team access: %q\n", address)

	importID := fmt.Sprintf("%s/%s/%s", organization, workspace.Name, access.ID)

	if err = tf.Import(ctx, address, importID, opts...); err != nil {
		return err
	}

	githubactions.Infof("Team access %q successfully imported\n", importID)

	return nil
}

// ImportRunTriggers imports all related inbound run triggers to the passed workspace
func ImportRunTriggers(ctx context.Context, tf TerraformCLI, triggers []*tfe.RunTrigger, client *tfe.Client, workspace *Workspace) error {
	if workspace.ID == nil {
		githubactions.Infof("Workspace %q not found, skipping run trigger import\n", workspace.Name)
		return nil
	}

	for _, trigger := range triggers {
		address := fmt.Sprintf("tfe_run_trigger.trigger[\"%s-%s\"]", workspace.Workspace, trigger.Sourceable.ID)

		imp, err := shouldImport(ctx, tf, address)
		if err != nil {
			return err
		}

		if !imp {
			githubactions.Infof("Run trigger %q already exists in state, skipping import\n", address)
			return nil
		}

		githubactions.Infof("Importing run trigger: %q\n", address)

		if err := tf.Import(ctx, address, trigger.ID); err != nil {
			return err
		}

		githubactions.Infof("Run trigger %q successfully imported\n", address)
	}

	return nil
}

// ImportWorkspaceResources discovers and imports resources related to the passed workspace
func ImportWorkspaceResources(ctx context.Context, client *tfe.Client, tf *tfexec.Terraform, filePath string, workspace *Workspace, organization string, providers []Provider) error {
	if workspace.ID == nil {
		githubactions.Infof("Workspace %q is not found, skipping import", workspace.Name)
		return nil
	}

	module := NewModule()

	wsConfig, err := NewWorkspaceResource(ctx, client, []*Workspace{workspace}, &WorkspaceResourceOptions{})
	if err != nil {
		return err
	}

	module.AppendResource("tfe_workspace", "workspace", wsConfig)

	variables, err := FetchRelatedVariables(ctx, client, workspace)
	if err != nil {
		return err
	}

	for _, variable := range variables {
		v := ToVariable(variable, workspace)

		module.AppendResource("tfe_variable", fmt.Sprintf("%s-%s", workspace.Workspace, v.Key), v.ToResource())
	}

	tfeTeams, err := FetchRelatedTeams(ctx, client, workspace, organization)
	if err != nil {
		return err
	}

	tfeTeamAccess, err := FetchRelatedTeamAccess(ctx, client, workspace)
	if err != nil {
		return err
	}

	teamAccess, err := ToTeamAccessItems(tfeTeamAccess, tfeTeams, workspace)
	if err != nil {
		return err
	}

	AppendTeamAccess(module, teamAccess, organization)

	tfeTriggers, err := FetchInboundRunTriggers(ctx, client, *workspace.ID)
	if err != nil {
		return err
	}

	AppendRunTriggers(module, ToRunTriggers(tfeTriggers, workspace))

	AddProviders(module, providers)

	if err := TerraformInit(ctx, tf, module, filePath); err != nil {
		return err
	}

	if err := ImportWorkspace(ctx, tf, client, workspace, organization); err != nil {
		return err
	}

	for _, variable := range variables {
		if err := ImportVariable(ctx, tf, variable, workspace, organization); err != nil {
			return err
		}
	}

	for _, access := range tfeTeamAccess {
		if err := ImportTeamAccess(ctx, tf, access, workspace, organization); err != nil {
			return err
		}
	}

	if err := ImportRunTriggers(ctx, tf, tfeTriggers, client, workspace); err != nil {
		return err
	}

	return nil
}

// ImportResources discovers and imports resources related to the passed workspaces
func ImportResources(ctx context.Context, client *tfe.Client, tf *tfexec.Terraform, module *tfconfig.Module, filePath string, workspaces []*Workspace, organization string, providers []Provider) error {
	for _, ws := range workspaces {
		if err := ImportWorkspaceResources(ctx, client, tf, filePath, ws, organization, providers); err != nil {
			return err
		}

		if err := TerraformInit(ctx, tf, module, filePath); err != nil {
			return err
		}
	}

	return nil
}
