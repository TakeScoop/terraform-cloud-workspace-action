package main

import (
	"context"
	"fmt"

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

func ImportWorkspace(ctx context.Context, tf *tfexec.Terraform, tfc *tfe.Client, name string, organization string, opts ...tfexec.ImportOption) error {
	address := fmt.Sprintf("tfe_workspace.workspace[%q]", name)

	imp, err := shouldImport(ctx, tf, address)
	if err != nil {
		return err
	}

	if !imp {
		fmt.Printf("Workspace %q already exists in state, skipping import\n", name)
		return nil
	}

	ws, err := tfc.Workspaces.Read(ctx, organization, name)
	if err != nil {
		return err
	}

	fmt.Printf("Importing workspace: %s\n", name)

	err = tf.Import(ctx, address, ws.ID, opts...)
	if err != nil {
		return err
	}

	fmt.Printf("Successful workspace import: %s\n", name)

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

	ws, err := client.Workspaces.Read(ctx, organization, workspace)
	if err != nil {
		return err
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
