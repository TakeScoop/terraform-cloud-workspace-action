package main

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
)

var maxPageSize int = 100

func shouldImport(ctx context.Context, tf *tfexec.Terraform, address string) (bool, error) {
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

func ImportWorkspace(ctx context.Context, tf *tfexec.Terraform, client *tfe.Client, name string, organization string, opts ...tfexec.ImportOption) error {
	address := fmt.Sprintf("tfe_workspace.workspace[%q]", name)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		fmt.Printf("Workspace %q already exists in state, skipping import\n", name)
		return nil
	}

	ws, err := GetWorkspaceIfExists(ctx, client, organization, name)
	if err != nil {
		return err
	}

	if ws == nil {
		fmt.Printf("Workspace %q not found, skipping import\n", name)
		return nil
	}

	fmt.Printf("Importing workspace: %s\n", ws.Name)

	err = tf.Import(ctx, address, ws.ID, opts...)
	if err != nil {
		return err
	}

	fmt.Printf("Successful workspace import: %s\n", ws.Name)

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

// GetWorkspaceIfExists returns the requested workspace, nil if the workspace does not exist, an error for any other issues fetching the workspace
func GetWorkspaceIfExists(ctx context.Context, client *tfe.Client, organization string, workspace string) (*tfe.Workspace, error) {
	ws, err := client.Workspaces.Read(ctx, organization, workspace)
	if err != nil {
		if err.Error() == "resource not found" {
			return nil, nil
		}
		return nil, err
	}

	return ws, nil
}

func ImportVariable(ctx context.Context, tf *tfexec.Terraform, client *tfe.Client, key string, workspace string, organization string, opts ...tfexec.ImportOption) error {
	address := fmt.Sprintf("tfe_variable.%s-%s", workspace, key)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		fmt.Printf("Variable %q already exists in state, skipping import\n", address)
		return nil
	}

	fmt.Printf("Importing variable: %q\n", address)

	ws, err := GetWorkspaceIfExists(ctx, client, organization, workspace)
	if err != nil {
		return err
	}

	if ws == nil {
		fmt.Printf("Workspace %q not found, skipping import\n", workspace)
		return nil
	}

	v, err := fetchVariableByKey(ctx, client, key, ws.ID, 1)
	if err != nil {
		return err
	}

	if v == nil {
		fmt.Printf("Variable %q for workspace %q not found, skipping import\n", key, workspace)
		return nil
	}

	importID := fmt.Sprintf("%s/%s/%s", organization, workspace, v.ID)

	err = tf.Import(ctx, address, importID, opts...)
	if err != nil {
		return err
	}

	fmt.Printf("Variable %q successfully imported\n", importID)

	return nil
}

// ImportTeamAccess imports a team access resource by looking up an existing relation
func ImportTeamAccess(ctx context.Context, tf *tfexec.Terraform, client *tfe.Client, organization string, workspace string, teamID string, opts ...tfexec.ImportOption) error {
	if teamID == "" {
		fmt.Println("Skipping team access import, required team ID was not passed")
		return nil
	}

	if !strings.HasPrefix(teamID, "team-") {
		return fmt.Errorf("team ID passed for team access import, but it was not of the static format team-xxx: %s", teamID)
	}

	address := fmt.Sprintf("tfe_team_access.teams[\"%s-%s\"]", workspace, teamID)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		fmt.Printf("Team access %q already exists in state, skipping import\n", address)
		return nil
	}

	ws, err := GetWorkspaceIfExists(ctx, client, organization, workspace)
	if err != nil {
		return err
	}

	if ws == nil {
		fmt.Printf("Workspace %q not found, skipping import\n", workspace)
		return nil
	}

	fmt.Printf("Importing team access: %q\n", address)

	teamAccess, err := client.TeamAccess.List(ctx, tfe.TeamAccessListOptions{
		WorkspaceID: &ws.ID,
	})
	if err != nil {
		return err
	}

	var teamAccessID string

	for _, access := range teamAccess.Items {
		if access.Team.ID == teamID {
			teamAccessID = access.ID
		}
	}

	if teamAccessID == "" {
		fmt.Printf("Team access %q for workspace %q not found, skipping import\n", teamID, workspace)
		return nil
	}

	importID := fmt.Sprintf("%s/%s/%s", organization, workspace, teamAccessID)

	if err = tf.Import(ctx, address, importID, opts...); err != nil {
		return err
	}

	fmt.Printf("Team access %q successfully imported\n", importID)

	return nil
}
