package action

import (
	"context"
	"fmt"
	"os"
	"strings"
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

func SetTestEnvs(envs map[string]string) {
	for key, value := range envs {
		os.Setenv(fmt.Sprintf("INPUT_%s", strings.ToUpper(key)), value)
	}
}

func UnsetTestEnvs(envs map[string]string) {
	for key := range envs {
		os.Unsetenv(fmt.Sprintf("INPUT_%s", strings.ToUpper(key)))
	}
}

func TestCreateWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	envs := map[string]string{
		"terraform_token":          os.Getenv("tf_token"),
		"terraform_organization":   "ryanwholey",
		"terraform_host":           "app.terraform.io",
		"name":                     fmt.Sprintf("%s-%d", workspacePrefix, time.Now().Unix()),
		"import":                   "true",
		"apply":                    "true",
		"tfe_provider_version":     "0.25.3",
		"runner_terraform_version": "1.0.5",
		"terraform_version":        "1.0.5",
	}

	SetTestEnvs(envs)

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", envs["terraform_host"]),
		Token:   envs["terraform_token"],
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := RemoveTestWorkspaces(ctx, client, envs["terraform_organization"], workspacePrefix); err != nil {
		t.Fatal(err)
	}

	_, err = client.Workspaces.Read(ctx, envs["terraform_organization"], envs["name"])
	if err == nil {
		t.Fatal("workspace should not exist, and an error should be returned")
	}

	if err.Error() != "resource not found" {
		t.Fatalf("Error is not workspace not found: %s", err)
	}

	Run()

	ws, err := client.Workspaces.Read(ctx, envs["terraform_organization"], envs["name"])
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ws.Name, envs["name"])

	t.Cleanup(func() {
		UnsetTestEnvs(envs)

		if err := RemoveTestWorkspaces(ctx, client, envs["terraform_organization"], workspacePrefix); err != nil {
			t.Fatal(err)
		}
	})
}

func TestImportExistingResources(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	envs := map[string]string{
		"terraform_token":          os.Getenv("tf_token"),
		"terraform_organization":   "ryanwholey",
		"terraform_host":           "app.terraform.io",
		"name":                     fmt.Sprintf("%s-%d", workspacePrefix, time.Now().Unix()),
		"import":                   "true",
		"apply":                    "true",
		"tfe_provider_version":     "0.25.3",
		"runner_terraform_version": "1.0.5",
		"terraform_version":        "1.0.5",
		"variables": `---
- key: foo
	value: baz
	category: terraform`,
	}

	SetTestEnvs(envs)

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", envs["terraform_host"]),
		Token:   envs["terraform_token"],
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := RemoveTestWorkspaces(ctx, client, envs["terraform_organization"], workspacePrefix); err != nil {
		t.Fatal(err)
	}

	ws, err := client.Workspaces.Create(ctx, envs["terraform_organization"], tfe.WorkspaceCreateOptions{
		Name:             strPtr(envs["name"]),
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

	assert.Equal(t, v.Category, *tfe.Category(tfe.CategoryEnv))

	Run()

	ws, err = client.Workspaces.Read(ctx, envs["terraform_organization"], envs["name"])
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
		UnsetTestEnvs(envs)

		if err := RemoveTestWorkspaces(ctx, client, envs["terraform_organization"], workspacePrefix); err != nil {
			t.Fatal(err)
		}
	})
}
