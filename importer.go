package main

import (
	"context"
	"fmt"
	"log"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type Importer interface {
	ImportWorkspace(ctx context.Context, name string, organization string, opts ...tfexec.ImportOption) error
}

type ImporterClient struct {
	e *tfexec.Terraform
	c *tfe.Client
}

func NewImporter(tf *tfexec.Terraform, token string, host string) (*ImporterClient, error) {
	tfc, err := tfe.NewClient(&tfe.Config{
		Token:   token,
		Address: fmt.Sprintf("https://%s", host),
	})

	if err != nil {
		return nil, err
	}

	return &ImporterClient{
		tf,
		tfc,
	}, nil
}

func (tf *ImporterClient) ImportWorkspace(ctx context.Context, name string, organization string, opts ...tfexec.ImportOption) error {
	address := fmt.Sprintf("tfe_workspace.workspace[\"%s\"]", name)

	state, err := tf.e.Show(context.Background())
	if err != nil {
		log.Fatalf("Failed to read state: %s", err)
	}

	found := false
	if state.Values == nil {
		fmt.Println("No state found, importing")
	} else {
		for _, r := range state.Values.RootModule.Resources {
			if address == r.Address {
				found = true
				break
			}
		}
	}

	if found {
		fmt.Printf("Workspace '%s' already exists in state\n", name)
		return nil
	}

	ws, err := tf.c.Workspaces.Read(context.Background(), organization, name)
	if err != nil {
		fmt.Printf("Could not read workspace '%s': %s\n", name, err)
	} else {
		fmt.Printf("Importing workspace: %s\n", name)

		err = tf.e.Import(
			context.Background(),
			address,
			ws.ID,
			opts...,
		)
		if err != nil {
			return err
		}

		fmt.Printf("Successful workspace import: %s\n", name)
	}

	return nil
}
