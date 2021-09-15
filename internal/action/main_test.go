package action

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

var testWorkspacePrefix string = "action-test"

// newTestInputs returns an Inputs object with test defaults
func newTestInputs(t *testing.T) *Inputs {
	action, err := getActionConfig()
	if err != nil {
		t.Fatal(err)
	}

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

// removeTestWorkspacesFunc returns a function that removes all workspaces created by the integration tests
func removeTestWorkspacesFunc(t *testing.T, ctx context.Context, client *tfe.Client) func() {
	return func() {
		removeTestWorkspaces(t, ctx, client)
	}
}

// removeTestWorkspaces deletes all test workspaces created by these tests
func removeTestWorkspaces(t *testing.T, ctx context.Context, client *tfe.Client) {
	workspaces, err := client.Workspaces.List(ctx, os.Getenv("TF_ORGANIZATION"), tfe.WorkspaceListOptions{
		Search: tfe.String(testWorkspacePrefix),
		ListOptions: tfe.ListOptions{
			PageSize: maxPageSize,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, ws := range workspaces.Items {
		if err := client.Workspaces.DeleteByID(ctx, ws.ID); err != nil {
			t.Fatal(err)
		}
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
func getActionConfig() (*ActionConfig, error) {
	actionFile, err := ioutil.ReadFile("../../action.yml")
	if err != nil {
		return nil, err
	}

	var action ActionConfig

	if err := yaml.Unmarshal(actionFile, &action); err != nil {
		return nil, err
	}

	return &action, nil
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
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client))

	removeTestWorkspaces(t, ctx, client)

	_, err = client.Workspaces.Read(ctx, inputs.Organization, inputs.Name)
	if err == nil {
		t.Fatal("workspace should not exist, and an error should be returned")
	}

	assert.ErrorIs(t, err, tfe.ErrResourceNotFound)

	if err = Run(inputs); err != nil {
		t.Fatal(err)
	}

	ws, err := client.Workspaces.Read(ctx, inputs.Organization, inputs.Name)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client))

	removeTestWorkspaces(t, ctx, client)

	ws, err := client.Workspaces.Create(ctx, inputs.Organization, tfe.WorkspaceCreateOptions{
		Name:             &inputs.Name,
		TerraformVersion: tfe.String("1.0.0"),
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ws.TerraformVersion, "1.0.0")

	v, err := client.Variables.Create(ctx, ws.ID, tfe.VariableCreateOptions{
		Key:      tfe.String("foo"),
		Value:    tfe.String("bar"),
		Category: tfe.Category(tfe.CategoryTerraform),
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, v.Value, "bar")

	if err = Run(inputs); err != nil {
		t.Fatal(err)
	}

	ws, err = client.Workspaces.Read(ctx, inputs.Organization, inputs.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ws.TerraformVersion, inputs.TerraformVersion)

	v, err = client.Variables.Read(ctx, ws.ID, v.ID)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client))

	removeTestWorkspaces(t, ctx, client)

	ws, err := client.Workspaces.Create(ctx, inputs.Organization, tfe.WorkspaceCreateOptions{
		Name:             &inputs.Name,
		TerraformVersion: &inputs.TerraformVersion,
	})
	if err != nil {
		t.Fatal(err)
	}

	v, err := client.Variables.Create(ctx, ws.ID, tfe.VariableCreateOptions{
		Key:      tfe.String("foo"),
		Value:    tfe.String("bar"),
		Category: tfe.Category(tfe.CategoryTerraform),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = Run(inputs); err != nil {
		t.Fatal(err)
	}

	_, err = client.Variables.Read(ctx, ws.ID, v.ID)
	if err == nil {
		t.Fatal("Expected variable not to exist")
	}

	assert.ErrorIs(t, err, tfe.ErrResourceNotFound)
}

func TestMultipleWorkspaces(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	inputs := newTestInputs(t)

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", inputs.Host),
		Token:   inputs.Token,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client))

	removeTestWorkspaces(t, ctx, client)

	ws, err := client.Workspaces.List(ctx, inputs.Organization, tfe.WorkspaceListOptions{
		Search: &testWorkspacePrefix,
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, ws.Items, 0)

	if err = Run(inputs); err != nil {
		t.Fatal(err)
	}

	ws, err = client.Workspaces.List(ctx, inputs.Organization, tfe.WorkspaceListOptions{
		Search: &testWorkspacePrefix,
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, ws.Items, 2)

	for _, ws := range ws.Items {
		v, err := client.Variables.List(ctx, ws.ID, tfe.VariableListOptions{})
		if err != nil {
			t.Fatal(err)
		}

		assert.Len(t, v.Items, 1)
	}
}

// fuzzyFindWorksapce finds the first workspace matching the passed match string
func fuzzyFindWorksapce(match string, workspaceList *tfe.WorkspaceList) *tfe.Workspace {
	for _, ws := range workspaceList.Items {
		if strings.Contains(ws.Name, match) {
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
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(removeTestWorkspacesFunc(t, ctx, client))
	removeTestWorkspaces(t, ctx, client)

	wsSourceAll, err := client.Workspaces.Create(ctx, inputs.Organization, tfe.WorkspaceCreateOptions{
		Name:             tfe.String(fmt.Sprintf("%s-source-all-%d", testWorkspacePrefix, time.Now().Unix())),
		TerraformVersion: tfe.String("1.0.0"),
	})
	assert.NoError(t, err)

	wsSourceAlpha, err := client.Workspaces.Create(ctx, inputs.Organization, tfe.WorkspaceCreateOptions{
		Name:             tfe.String(fmt.Sprintf("%s-source-single-%d", testWorkspacePrefix, time.Now().Unix())),
		TerraformVersion: tfe.String("1.0.0"),
	})
	assert.NoError(t, err)

	inputs.RunTriggers = fmt.Sprintf("- %s", wsSourceAll.ID)
	inputs.WorkspaceRunTriggers = fmt.Sprintf(`---
alpha:
  - %s
`, wsSourceAlpha.ID)

	err = Run(inputs)
	assert.NoError(t, err)

	workspaces, err := client.Workspaces.List(ctx, inputs.Organization, tfe.WorkspaceListOptions{
		Search: &testWorkspacePrefix,
	})
	assert.NoError(t, err)

	assert.Len(t, workspaces.Items, 4)

	alpha := fuzzyFindWorksapce("alpha", workspaces)
	if alpha == nil {
		t.Fatal("alpha workspace not found")
	}

	triggers, err := client.RunTriggers.List(ctx, alpha.ID, tfe.RunTriggerListOptions{
		RunTriggerType: tfe.String("inbound"),
	})
	assert.NoError(t, err)

	assert.Len(t, triggers.Items, 2)

	beta := fuzzyFindWorksapce("beta", workspaces)
	if beta == nil {
		t.Fatal("beta workspace not found")
	}

	triggers, err = client.RunTriggers.List(ctx, beta.ID, tfe.RunTriggerListOptions{
		RunTriggerType: tfe.String("inbound"),
	})
	assert.NoError(t, err)

	assert.Len(t, triggers.Items, 1)
}
