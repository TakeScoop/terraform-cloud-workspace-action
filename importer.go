package main

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
)

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
