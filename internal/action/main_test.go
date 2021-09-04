package action

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
)

var workspacePrefix string = "action-test"

func RemoveTestWorkspaces(ctx context.Context, client *tfe.Client, organization string, prefix string) error {
	workspaces, err := client.Workspaces.List(ctx, organization, tfe.WorkspaceListOptions{
		Search: &prefix,
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	})
	if err != nil {
		return err
	}

	for _, ws := range workspaces.Items {
		if err := client.Workspaces.DeleteByID(ctx, ws.ID); err != nil {
			return err
		}
	}

	return nil
}

func TestCreateWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	config := &RunConfig{
		Token:                  os.Getenv("tf_token"),
		Organization:           "ryanwholey",
		Host:                   "app.terraform.io",
		Name:                   fmt.Sprintf("%s-%d", workspacePrefix, time.Now().Unix()),
		Import:                 true,
		Apply:                  true,
		TFEProviderVersion:     "0.25.3",
		RunnerTerraformVersion: "1.0.5",
		TerraformVersion:       "1.0.5",
	}

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", config.Host),
		Token:   config.Token,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := RemoveTestWorkspaces(ctx, client, config.Organization, workspacePrefix); err != nil {
		t.Fatal(err)
	}

	_, err = client.Workspaces.Read(ctx, config.Organization, config.Name)
	if err == nil {
		t.Fatal("workspace should not exist, and an error should be returned")
	}

	if err.Error() != "resource not found" {
		t.Fatalf("Error is not workspace not found: %s", err)
	}

	Run(config)

	ws, err := client.Workspaces.Read(ctx, config.Organization, config.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ws.Name, config.Name)

	t.Cleanup(func() {
		if err := RemoveTestWorkspaces(ctx, client, config.Organization, workspacePrefix); err != nil {
			t.Fatal(err)
		}
	})
}

func TestImportExistingResources(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	config := &RunConfig{
		Token:                  os.Getenv("tf_token"),
		Organization:           "ryanwholey",
		Host:                   "app.terraform.io",
		Name:                   fmt.Sprintf("%s-%d", workspacePrefix, time.Now().Unix()),
		Import:                 true,
		Apply:                  true,
		TFEProviderVersion:     "0.25.3",
		RunnerTerraformVersion: "1.0.5",
		TerraformVersion:       "1.0.5",
		Variables: `---
- key: foo
  value: baz
  category: terraform`,
	}

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", config.Host),
		Token:   config.Token,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := RemoveTestWorkspaces(ctx, client, config.Organization, workspacePrefix); err != nil {
		t.Fatal(err)
	}

	ws, err := client.Workspaces.Create(ctx, config.Organization, tfe.WorkspaceCreateOptions{
		Name:             &config.Name,
		TerraformVersion: strPtr("1.0.4"),
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ws.TerraformVersion, "1.0.4")

	v, err := client.Variables.Create(ctx, ws.ID, tfe.VariableCreateOptions{
		Key:      strPtr("foo"),
		Value:    strPtr("bar"),
		Category: tfe.Category(tfe.CategoryTerraform),
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, v.Value, "bar")

	Run(config)

	ws, err = client.Workspaces.Read(ctx, config.Organization, config.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ws.TerraformVersion, "1.0.5")

	v, err = client.Variables.Read(ctx, ws.ID, v.ID)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, v.Value, "baz")

	t.Cleanup(func() {
		if err := RemoveTestWorkspaces(ctx, client, config.Organization, workspacePrefix); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDriftCorrection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	config := &RunConfig{
		Token:                  os.Getenv("tf_token"),
		Organization:           "ryanwholey",
		Host:                   "app.terraform.io",
		Name:                   fmt.Sprintf("%s-%d", workspacePrefix, time.Now().Unix()),
		Import:                 true,
		Apply:                  true,
		TFEProviderVersion:     "0.25.3",
		RunnerTerraformVersion: "1.0.5",
		TerraformVersion:       "1.0.5",
	}

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", config.Host),
		Token:   config.Token,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := RemoveTestWorkspaces(ctx, client, config.Organization, workspacePrefix); err != nil {
		t.Fatal(err)
	}

	ws, err := client.Workspaces.Create(ctx, config.Organization, tfe.WorkspaceCreateOptions{
		Name:             &config.Name,
		TerraformVersion: &config.TerraformVersion,
	})
	if err != nil {
		t.Fatal(err)
	}

	v, err := client.Variables.Create(ctx, ws.ID, tfe.VariableCreateOptions{
		Key:      strPtr("foo"),
		Value:    strPtr("bar"),
		Category: tfe.Category(tfe.CategoryTerraform),
	})
	if err != nil {
		t.Fatal(err)
	}

	Run(config)

	_, err = client.Variables.Read(ctx, ws.ID, v.ID)
	if err == nil {
		t.Fatal("Expected variable not to exist")
	}

	if err.Error() != "resource not found" {
		t.Fatalf("Expected error to be resource not found: %s", err)
	}

	t.Cleanup(func() {
		if err := RemoveTestWorkspaces(ctx, client, config.Organization, workspacePrefix); err != nil {
			t.Fatal(err)
		}
	})
}

// func TestMultipleWorkspaces(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test")
// 	}

// 	ctx := context.Background()

// 	envs := map[string]string{
// 		"terraform_token":        os.Getenv("tf_token"),
// 		"terraform_organization": "ryanwholey",
// 		"terraform_host":         "app.terraform.io",
// 		"name":                   fmt.Sprintf("%s-%d", workspacePrefix, time.Now().Unix()),
// 		"workspaces": `---
// - staging
// - production`,
// 		"workspace_variables": `---
// staging:
//   - key: environment
//     value: staging
//     category: env
// production:
//   - key: environment
//     value: production
//     category: env`,
// 		"import":                   "true",
// 		"apply":                    "true",
// 		"tfe_provider_version":     "0.25.3",
// 		"runner_terraform_version": "1.0.5",
// 		"terraform_version":        "1.0.5",
// 	}

// 	SetTestEnvs(envs)

// 	client, err := tfe.NewClient(&tfe.Config{
// 		Address: fmt.Sprintf("https://%s", envs["terraform_host"]),
// 		Token:   envs["terraform_token"],
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if err := RemoveTestWorkspaces(ctx, client, envs["terraform_organization"], workspacePrefix); err != nil {
// 		t.Fatal(err)
// 	}

// 	ws, err := client.Workspaces.List(ctx, envs["terraform_organization"], tfe.WorkspaceListOptions{
// 		Search: &workspacePrefix,
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assert.Equal(t, len(ws.Items), 0)

// 	Run()

// 	ws, err = client.Workspaces.List(ctx, envs["terraform_organization"], tfe.WorkspaceListOptions{
// 		Search: &workspacePrefix,
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assert.Equal(t, len(ws.Items), 2)

// 	for _, ws := range ws.Items {
// 		v, err := client.Variables.List(ctx, ws.ID, tfe.VariableListOptions{})
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		assert.Equal(t, len(v.Items), 1)
// 	}

// 	t.Cleanup(func() {
// 		UnsetTestEnvs(envs)

// 		if err := RemoveTestWorkspaces(ctx, client, envs["terraform_organization"], workspacePrefix); err != nil {
// 			t.Fatal(err)
// 		}
// 	})
// }
