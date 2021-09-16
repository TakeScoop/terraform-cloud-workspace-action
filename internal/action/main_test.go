package action

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

var testWorkspacePrefix string = "action-test"

// newTestInputs returns an Inputs object with test defaults
func newTestInputs(t *testing.T) *Inputs {
	action := getActionConfig(t)

	imp, err := strconv.ParseBool(action.Inputs["import"].Default)
	if err != nil {
		t.Fatal(err)
	}

	token := os.Getenv("TF_TOKEN")
	if token == "" {
		t.Fatal(`Error: "TF_TOKEN" must be set in the environment`)
	}

	organization := os.Getenv("TF_ORGANIZATION")
	if organization == "" {
		t.Fatal(`Error: "TF_ORGANIZATION" must be set in the environment`)
	}

	return &Inputs{
		Token:                  token,
		Organization:           organization,
		Host:                   action.Inputs["terraform_host"].Default,
		Name:                   fmt.Sprintf("%s-%d", testWorkspacePrefix, time.Now().Unix()),
		Import:                 imp,
		Apply:                  true,
		TFEProviderVersion:     action.Inputs["tfe_provider_version"].Default,
		RunnerTerraformVersion: action.Inputs["runner_terraform_version"].Default,
		TerraformVersion:       action.Inputs["terraform_version"].Default,
	}
}

// removeTestWorkspacesFunc returns a function that removes matching workspaces created by the integration tests
func removeTestWorkspacesFunc(t *testing.T, ctx context.Context, client *tfe.Client, match string) func() {
	return func() {
		removeTestWorkspaces(t, ctx, client, match)
	}
}

// removeTestWorkspaces deletes matching test workspaces created by the integration tests
func removeTestWorkspaces(t *testing.T, ctx context.Context, client *tfe.Client, match string) {
	workspaces, err := client.Workspaces.List(ctx, os.Getenv("TF_ORGANIZATION"), tfe.WorkspaceListOptions{
		Search: tfe.String(match),
		ListOptions: tfe.ListOptions{
			PageSize: maxPageSize,
		},
	})
	require.NoError(t, err)

	for _, ws := range workspaces.Items {
		err := client.Workspaces.DeleteByID(ctx, ws.ID)
		assert.NoError(t, err)
	}
}

type ActionInput struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
}

type ActionConfig struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Inputs      map[string]ActionInput `yaml:"inputs"`
}

// getActionConfig returns the action configuration file
func getActionConfig(t *testing.T) *ActionConfig {
	actionFile, err := ioutil.ReadFile("../../action.yml")
	require.NoError(t, err)

	var action ActionConfig

	err = yaml.Unmarshal(actionFile, &action)
	require.NoError(t, err)

	return &action
}

func TestCreateWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	inputs := newTestInputs(t)

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", inputs.Host),
		Token:   inputs.Token,
	})
	require.NoError(t, err)

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client, inputs.Name))

	_, err = client.Workspaces.Read(ctx, inputs.Organization, inputs.Name)
	assert.ErrorIs(t, err, tfe.ErrResourceNotFound)

	if err = Run(inputs); err != nil {
		require.NoError(t, err)
	}

	ws, err := client.Workspaces.Read(ctx, inputs.Organization, inputs.Name)
	require.NoError(t, err)

	assert.Equal(t, ws.Name, inputs.Name)
}

func TestImportExistingResources(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	inputs := newTestInputs(t)

	inputs.Variables = `---
- key: foo
  value: baz
  category: terraform`

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", inputs.Host),
		Token:   inputs.Token,
	})
	require.NoError(t, err)

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client, inputs.Name))

	ws, err := client.Workspaces.Create(ctx, inputs.Organization, tfe.WorkspaceCreateOptions{
		Name:             &inputs.Name,
		TerraformVersion: tfe.String("1.0.0"),
	})
	require.NoError(t, err)

	assert.Equal(t, ws.TerraformVersion, "1.0.0")

	v, err := client.Variables.Create(ctx, ws.ID, tfe.VariableCreateOptions{
		Key:      tfe.String("foo"),
		Value:    tfe.String("bar"),
		Category: tfe.Category(tfe.CategoryTerraform),
	})
	require.NoError(t, err)

	assert.Equal(t, v.Value, "bar")

	err = Run(inputs)
	require.NoError(t, err)

	ws, err = client.Workspaces.Read(ctx, inputs.Organization, inputs.Name)
	require.NoError(t, err)

	assert.Equal(t, ws.TerraformVersion, inputs.TerraformVersion)

	v, err = client.Variables.Read(ctx, ws.ID, v.ID)
	require.NoError(t, err)

	assert.Equal(t, v.Value, "baz")
}

func TestDriftCorrection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	inputs := newTestInputs(t)

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", inputs.Host),
		Token:   inputs.Token,
	})
	require.NoError(t, err)

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client, inputs.Name))

	ws, err := client.Workspaces.Create(ctx, inputs.Organization, tfe.WorkspaceCreateOptions{
		Name:             &inputs.Name,
		TerraformVersion: &inputs.TerraformVersion,
	})
	require.NoError(t, err)

	v, err := client.Variables.Create(ctx, ws.ID, tfe.VariableCreateOptions{
		Key:      tfe.String("foo"),
		Value:    tfe.String("bar"),
		Category: tfe.Category(tfe.CategoryTerraform),
	})
	require.NoError(t, err)

	err = Run(inputs)
	require.NoError(t, err)

	_, err = client.Variables.Read(ctx, ws.ID, v.ID)
	assert.ErrorIs(t, err, tfe.ErrResourceNotFound)
}

func TestMultipleWorkspaces(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	inputs := newTestInputs(t)

	inputs.Workspaces = `---
- staging
- production`

	inputs.WorkspaceVariables = `---
staging:
  - key: environment
    value: staging
    category: env
production:
  - key: environment
    value: production
    category: env`

	inputs.Tags = `---
- all`

	inputs.WorkspaceTags = `---
staging:
  - staging
production:
  - production`

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", inputs.Host),
		Token:   inputs.Token,
	})
	require.NoError(t, err)

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client, inputs.Name))

	ws, err := client.Workspaces.List(ctx, inputs.Organization, tfe.WorkspaceListOptions{
		Search: &testWorkspacePrefix,
	})
	require.NoError(t, err)

	assert.Len(t, ws.Items, 0)

	err = Run(inputs)
	require.NoError(t, err)

	ws, err = client.Workspaces.List(ctx, inputs.Organization, tfe.WorkspaceListOptions{
		Search: &testWorkspacePrefix,
	})
	require.NoError(t, err)

	assert.Len(t, ws.Items, 2)

	for _, ws := range ws.Items {
		v, err := client.Variables.List(ctx, ws.ID, tfe.VariableListOptions{})
		require.NoError(t, err)

		assert.Len(t, v.Items, 1)
	}
}

// findWorkspaceByName finds the first workspace matching the passed match string
func findWorkspaceByName(name string, workspaceList *tfe.WorkspaceList) *tfe.Workspace {
	for _, ws := range workspaceList.Items {
		if ws.Name == name {
			return ws
		}
	}

	return nil
}

func TestWorkspaceRunTriggers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	inputs := newTestInputs(t)

	inputs.Workspaces = `---
- alpha
- beta`

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", inputs.Host),
		Token:   inputs.Token,
	})
	require.NoError(t, err)

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client, inputs.Name))

	wsSourceAll, err := client.Workspaces.Create(ctx, inputs.Organization, tfe.WorkspaceCreateOptions{
		Name:             tfe.String(fmt.Sprintf("%s-source-all-%d", inputs.Name, time.Now().Unix())),
		TerraformVersion: tfe.String("1.0.0"),
	})
	require.NoError(t, err)

	wsSourceAlpha, err := client.Workspaces.Create(ctx, inputs.Organization, tfe.WorkspaceCreateOptions{
		Name:             tfe.String(fmt.Sprintf("%s-source-single-%d", inputs.Name, time.Now().Unix())),
		TerraformVersion: tfe.String("1.0.0"),
	})
	require.NoError(t, err)

	inputs.RunTriggers = fmt.Sprintf("- id: %s", wsSourceAll.ID)
	inputs.WorkspaceRunTriggers = fmt.Sprintf("alpha: [id: %s]", wsSourceAlpha.ID)

	err = Run(inputs)
	require.NoError(t, err)

	workspaces, err := client.Workspaces.List(ctx, inputs.Organization, tfe.WorkspaceListOptions{
		Search: &inputs.Name,
	})
	require.NoError(t, err)

	assert.Len(t, workspaces.Items, 4)

	alpha := findWorkspaceByName(fmt.Sprintf("%s-alpha", inputs.Name), workspaces)
	if alpha == nil {
		t.Fatal("alpha workspace not found")
	}

	triggers, err := client.RunTriggers.List(ctx, alpha.ID, tfe.RunTriggerListOptions{
		RunTriggerType: tfe.String("inbound"),
	})
	require.NoError(t, err)

	assert.Len(t, triggers.Items, 2)

	beta := findWorkspaceByName(fmt.Sprintf("%s-beta", inputs.Name), workspaces)
	if beta == nil {
		t.Fatal("beta workspace not found")
	}

	triggers, err = client.RunTriggers.List(ctx, beta.ID, tfe.RunTriggerListOptions{
		RunTriggerType: tfe.String("inbound"),
	})
	require.NoError(t, err)

	assert.Len(t, triggers.Items, 1)
}
