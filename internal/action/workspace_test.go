package action

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfconfig"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

var basicOauthClientResponse string = `
{
	"data": [
		{
			"id": "oc-sdlkfjdskljfsd",
			"type": "oauth-clients",
			"attributes": {
				"name": "github.com",
				"created-at": "2021-04-12T21:14:17.245Z",
				"callback-url": "https://app.terraform.io/auth/12345/callback",
				"connect-path": "/auth/12345?organization_id=12345",
				"service-provider": "github",
				"service-provider-display-name": "GitHub",
				"http-url": "https://github.com",
				"api-url": "https://api.github.com",
				"key": "12345",
				"secret": null,
				"rsa-public-key": null
			},
			"relationships": {
				"organization": {
					"data": {
						"id": "org",
						"type": "organizations"
					},
					"links": {
						"related": "/api/v2/organizations/org"
					}
				},
				"oauth-tokens": {
					"data": [
						{
							"id": "ot-678910",
							"type": "oauth-tokens"
						}
					],
					"links": {
						"related": "/api/v2/oauth-clients/oc-sdlkfjdskljfsd/oauth-tokens"
					}
				}
			}
		}
	]
}
`

func boolPtr(b bool) *bool {
	return &b
}

func TestGetVCSTokenIDByClientType(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	defer server.Close()

	mux.HandleFunc("/api/v2/organizations/org/oauth-clients", testServerResHandler(t, 200, basicOauthClientResponse))

	client := newTestTFClient(t, server.URL)

	t.Run("get client token ID by type", func(t *testing.T) {
		tokenID, err := GetVCSTokenIDByClientType(ctx, client, "org", "github")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, tokenID, "ot-678910")
	})
}

func TestWorkspaceJSONRender(t *testing.T) {
	t.Run("no VCS block added when VCSRepo is nil", func(t *testing.T) {
		b, err := json.MarshalIndent(tfeprovider.Workspace{
			ForEach: map[string]*tfeprovider.Workspace{
				"foo-staging":    {Name: "foo-staging"},
				"foo-production": {Name: "foo-production"},
			},
			Name:         "${each.value.name}",
			Organization: "org",
			AutoApply:    boolPtr(true),
			VCSRepo:      nil,
		}, "", "\t")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, string(b), `{
	"for_each": {
		"foo-production": {
			"name": "foo-production"
		},
		"foo-staging": {
			"name": "foo-staging"
		}
	},
	"auto_apply": true,
	"name": "${each.value.name}",
	"organization": "org"
}`)
	})
}

func TestNewWorkspaceResource(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	defer server.Close()

	mux.HandleFunc("/api/v2/organizations/org/oauth-clients", testServerResHandler(t, 200, basicOauthClientResponse))

	client := newTestTFClient(t, server.URL)

	type BoolTest struct {
		AutoApply *bool `json:"auto_apply,omitempty"`
	}

	t.Run("should render a basic workspace without unprovided values", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{
			{Name: "foo"},
		}, &WorkspaceResourceOptions{
			Organization: "org",
		})
		if err != nil {
			t.Fatal(err)
		}

		s, err := json.MarshalIndent(ws, "", "\t")

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, string(s), `{
	"for_each": {
		"foo": {
			"name": "foo"
		}
	},
	"name": "${each.value.name}",
	"organization": "org"
}`)
	})

	t.Run("should set boolean value to false if passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization: "org",
			AutoApply:    boolPtr(false),
		})
		if err != nil {
			t.Fatal(err)
		}

		b, err := json.Marshal(ws)
		if err != nil {
			t.Fatal(err)
		}

		bt := BoolTest{}

		if err := json.Unmarshal(b, &bt); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, *bt.AutoApply, false)
	})

	t.Run("should set boolean value to true if passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization: "org",
			AutoApply:    boolPtr(true),
		})
		if err != nil {
			t.Fatal(err)
		}

		b, err := json.Marshal(ws)
		if err != nil {
			t.Fatal(err)
		}

		bt := BoolTest{}

		if err := json.Unmarshal(b, &bt); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, *bt.AutoApply, true)
	})

	t.Run("should set boolean value nil if not passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization: "org",
		})
		if err != nil {
			t.Fatal(err)
		}

		b, err := json.Marshal(ws)
		if err != nil {
			t.Fatal(err)
		}

		bt := BoolTest{}
		if err := json.Unmarshal(b, &bt); err != nil {
			t.Fatal(err)
		}

		var nilBool *bool = nil
		assert.Equal(t, bt.AutoApply, nilBool)
	})

	t.Run("add VCS block type if VCS type is passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization: "org",
			VCSType:      "github",
			VCSRepo:      "org/repo",
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, ws.VCSRepo.OauthTokenID, "ot-678910")
		assert.Equal(t, ws.VCSRepo.IngressSubmodules, false)
		assert.Equal(t, ws.VCSRepo.Identifier, "org/repo")
	})

	t.Run("fail if vcs_repo is not passed", func(t *testing.T) {
		_, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization: "org",
			VCSType:      "github",
		})
		assert.EqualError(t, err, "VCS repository must be passed if VCS type or a VCS token ID is passed")
	})

	t.Run("use VCSTokenID directly when passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization: "org",
			VCSTokenID:   "TOKEN",
			VCSType:      "github",
			VCSRepo:      "org/repo",
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, ws.VCSRepo.OauthTokenID, "TOKEN")
	})

	t.Run("add AgentPoolID and ExecutionMode: \"agent\" when AgentPoolID is passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization: "org",
			AgentPoolID:  "12345",
		})
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, ws.AgentPoolID, "12345")
		assert.Equal(t, ws.ExecutionMode, "agent")
	})

	t.Run("add RemoteConsumerIDs and GlobalRemoteState if global_remote_state is false", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization:           "org",
			GlobalRemoteState:      boolPtr(false),
			RemoteStateConsumerIDs: "123,456,789",
		})
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, *ws.GlobalRemoteState, false)
		assert.Equal(t, ws.RemoteStateConsumerIDs, []string{"123", "456", "789"})
	})

	t.Run("add no remote IDs when none are passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization:      "org",
			GlobalRemoteState: boolPtr(false),
		})
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, *ws.GlobalRemoteState, false)
		assert.Equal(t, ws.RemoteStateConsumerIDs, []string{})
	})

	t.Run("add a description if passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, []*Workspace{{Name: "foo"}}, &WorkspaceResourceOptions{
			Organization: "org",
			Description:  "description",
		})
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, ws.Description, "description")
	})
}

func TestAppendTeamAccess(t *testing.T) {
	t.Run("Add basic team access", func(t *testing.T) {
		module := NewModule()

		AppendTeamAccess(module, TeamAccess{
			TeamAccessItem{TeamName: "Readers", Access: "read", Workspace: &Workspace{Name: "workspace"}},
			TeamAccessItem{TeamName: "Writers", Access: "write", Workspace: &Workspace{Name: "workspace"}},
		}, "org")

		assert.Equal(t, module.Data["tfe_team"]["teams"], TeamDataResource{
			ForEach: map[string]TeamDataResource{
				"Readers": {
					Name:         "Readers",
					Organization: "org",
				},
				"Writers": {
					Name:         "Writers",
					Organization: "org",
				},
			},
			Name:         "${each.value.name}",
			Organization: "${each.value.organization}",
		})

		assert.Equal(t, module.Resources["tfe_team_access"]["teams"], tfeprovider.TeamAccess{
			ForEach: map[string]tfeprovider.TeamAccess{
				"workspace-${data.tfe_team.teams[\"Writers\"].id}": {
					TeamID:      "${data.tfe_team.teams[\"Writers\"].id}",
					WorkspaceID: "${tfe_workspace.workspace[\"workspace\"].id}",
					Access:      "write",
				},
				"workspace-${data.tfe_team.teams[\"Readers\"].id}": {
					TeamID:      "${data.tfe_team.teams[\"Readers\"].id}",
					WorkspaceID: "${tfe_workspace.workspace[\"workspace\"].id}",
					Access:      "read",
				},
			},
			TeamID:      "${each.value.team_id}",
			WorkspaceID: "${each.value.workspace_id}",
			Access:      "${lookup(each.value, \"access\", null)}",
			DynamicPermissions: &tfeprovider.DynamicPermissions{
				Permission: []tfeprovider.DynamicPermissionEntry{{
					ForEach: "${lookup(each.value ,\"permissions\", null) != null ? {once: true} : {}}",
					Content: &tfeprovider.TeamAccessPermissions{
						Runs:             "${each.value.permissions.runs}",
						Variables:        "${each.value.permissions.variables}",
						StateVersions:    "${each.value.permissions.state_versions}",
						SentinelMocks:    "${each.value.permissions.sentinel_mocks}",
						WorkspaceLocking: "${each.value.permissions.workspace_locking}",
					},
				}},
			},
		})
	})

	t.Run("Add with permissions block", func(t *testing.T) {
		module := NewModule()

		AppendTeamAccess(module, TeamAccess{
			{TeamName: "Readers", Workspace: &Workspace{Name: "workspace"}, Permissions: &TeamAccessPermissionsInput{
				Runs:             "read",
				Variables:        "read",
				StateVersions:    "none",
				SentinelMocks:    "none",
				WorkspaceLocking: true,
			}},
		}, "org")

		assert.Equal(t, module.Data["tfe_team"]["teams"].(TeamDataResource).ForEach, map[string]TeamDataResource{
			"Readers": {
				Name:         "Readers",
				Organization: "org",
			},
		})

		assert.Equal(t, module.Resources["tfe_team_access"]["teams"].(tfeprovider.TeamAccess).ForEach, map[string]tfeprovider.TeamAccess{
			"workspace-${data.tfe_team.teams[\"Readers\"].id}": {
				TeamID:      "${data.tfe_team.teams[\"Readers\"].id}",
				WorkspaceID: "${tfe_workspace.workspace[\"workspace\"].id}",
				Access:      "",
				Permissions: &tfeprovider.TeamAccessPermissions{
					Runs:             "read",
					Variables:        "read",
					StateVersions:    "none",
					SentinelMocks:    "none",
					WorkspaceLocking: true,
				},
			},
		})
	})
}

func TestAddProviders(t *testing.T) {
	module := NewModule()

	AddProviders(module, []Provider{
		{Name: "tfe", Version: "0.25.0", Source: "hashicorp/tfe", Config: tfeprovider.Config{Hostname: "app.terraform.io"}},
	})

	assert.Equal(t, module.Providers["tfe"].(tfeprovider.Config).Hostname, "app.terraform.io")
	assert.Equal(t, module.Terraform.RequiredProviders["tfe"].Source, "hashicorp/tfe")
	assert.Equal(t, module.Terraform.RequiredProviders["tfe"].Version, "0.25.0")
}

func RunValidate(ctx context.Context, name string, tfexecPath string, module *tfconfig.Module) (*tfjson.ValidateOutput, error) {
	b, err := json.MarshalIndent(module, "", "\t")
	if err != nil {
		return nil, err
	}

	workDir, err := ioutil.TempDir("", name)
	if err != nil {
		return nil, err
	}

	if err = ioutil.WriteFile(path.Join(workDir, "main.tf.json"), b, 0644); err != nil {
		return nil, err
	}

	tf, err := tfexec.NewTerraform(workDir, tfexecPath)
	if err != nil {
		return nil, err
	}

	if err = tf.Init(ctx); err != nil {
		return nil, err
	}

	return tf.Validate(ctx)
}

func TestNewWorkspaceConfig(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	defer server.Close()

	mux.HandleFunc("/api/v2/organizations/org/oauth-clients", testServerResHandler(t, 200, basicOauthClientResponse))

	client := newTestTFClient(t, server.URL)

	tmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	execPath, err := tfinstall.Find(
		ctx,
		tfinstall.ExactVersion("1.0.3", tmpDir),
	)
	if err != nil {
		t.Fatal(err)
	}

	name := "test-repo"

	t.Run("validate basic workspace config", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client, []*Workspace{{Name: "foo"}}, &NewWorkspaceConfigOptions{
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		output, err := RunValidate(ctx, name, execPath, wsConfig)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, output.Valid, true, output.Diagnostics)
	})

	t.Run("validate with multiple workspaces", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client,
			[]*Workspace{
				{Name: "foo-staging"},
				{Name: "foo-production"},
			},
			&NewWorkspaceConfigOptions{
				Backend: map[string]interface{}{
					"local": map[string]interface{}{
						"path": "foo/terraform.tfstate",
					},
				},
				WorkspaceResourceOptions: &WorkspaceResourceOptions{
					Organization: "org",
				},
			})
		if err != nil {
			t.Fatal(err)
		}

		output, err := RunValidate(ctx, name, execPath, wsConfig)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, output.Valid, true, output.Diagnostics)
	})

	t.Run("validate using a passed backend", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client, []*Workspace{{Name: "foo"}}, &NewWorkspaceConfigOptions{
			Backend: map[string]interface{}{
				"local": map[string]interface{}{
					"path": "foo/terraform.tfstate",
				},
			},
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		output, err := RunValidate(ctx, name, execPath, wsConfig)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, output.Valid, true, output.Diagnostics)
	})

	t.Run("validate workspace with passed providers", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client, []*Workspace{{Name: "foo"}}, &NewWorkspaceConfigOptions{
			Providers: []Provider{
				{
					Name:    "tfe",
					Version: "0.25.0",
					Source:  "hashicorp/tfe",
					Config: tfeprovider.Config{
						Hostname: "app.terraform.io",
					},
				},
			},
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		output, err := RunValidate(ctx, name, execPath, wsConfig)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, output.Valid, true, output.Diagnostics)
	})

	t.Run("validate workspace with remote states", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client, []*Workspace{{Name: "foo"}}, &NewWorkspaceConfigOptions{
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
			RemoteStates: map[string]tfconfig.RemoteState{
				"foo": {
					Backend: "s3",
					Config: tfconfig.RemoteStateBackendConfig{
						Key:    "key",
						Bucket: "bucket",
						Region: "us-east-1",
					},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		output, err := RunValidate(ctx, name, execPath, wsConfig)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, output.Valid, true, output.Diagnostics)
	})

	t.Run("validate workspace with team access", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client, []*Workspace{{Name: "foo"}}, &NewWorkspaceConfigOptions{
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
			RemoteStates: map[string]tfconfig.RemoteState{
				"teams": {
					Backend: "remote",
					Config: tfconfig.RemoteStateBackendConfig{
						Organization: "org",
						Hostname:     "app.terraform.io",
						Workspaces: &tfconfig.RemoteStateBackendConfigWorkspaces{
							Name: "teams",
						},
					},
				},
			},
			TeamAccess: TeamAccess{
				TeamAccessItem{TeamName: "Readers", Workspace: &Workspace{Name: name}, Access: "read"},
				TeamAccessItem{TeamName: "Writers", Workspace: &Workspace{Name: name}, Permissions: &TeamAccessPermissionsInput{
					Runs:             "read",
					Variables:        "read",
					StateVersions:    "read",
					SentinelMocks:    "none",
					WorkspaceLocking: true,
				}},
				TeamAccessItem{TeamName: "${data.terraform_remote_state.teams.outputs.team}", Workspace: &Workspace{Name: name}, Access: "read"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		output, err := RunValidate(ctx, name, execPath, wsConfig)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, output.Valid, true, output.Diagnostics)
	})

	t.Run("validate workspace with variables", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client, []*Workspace{{Name: "foo"}}, &NewWorkspaceConfigOptions{
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
			Variables: Variables{
				Variable{
					Key:       "foo",
					Value:     "bar",
					Category:  "env",
					Workspace: &Workspace{Name: name},
				},
				Variable{
					Key:       "baz",
					Value:     "woz",
					Category:  "env",
					Workspace: &Workspace{Name: name},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		output, err := RunValidate(ctx, name, execPath, wsConfig)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, output.Valid, true, output.Diagnostics)
	})
}

func TestWillDestroy(t *testing.T) {
	t.Run("return true when a resource is scheduled for deletion", func(t *testing.T) {
		ctx := context.Background()

		workDir, err := ioutil.TempDir("", "deletion")
		if err != nil {
			t.Fatal(err)
		}

		defer os.RemoveAll(workDir)

		tf, err := NewTerraformExec(ctx, workDir, "1.0.3")
		if err != nil {
			t.Fatal(err)
		}

		b := []byte(`
	resource "random_pet" "first" {}
	resource "random_pet" "second" {}
	`)

		if err = ioutil.WriteFile(path.Join(workDir, "main.tf"), b, 0644); err != nil {
			t.Fatal(err)
		}

		if err = tf.Init(ctx); err != nil {
			t.Fatal(err)
		}

		if err = tf.Apply(ctx); err != nil {
			t.Fatal(err)
		}

		b = []byte(`
	resource "random_pet" "first" {}
	`)
		if err = ioutil.WriteFile(path.Join(workDir, "main.tf"), b, 0644); err != nil {
			t.Fatal(err)
		}

		planPath := "plan.txt"

		if _, err = tf.Plan(ctx, tfexec.Out(planPath)); err != nil {
			t.Fatal(err)
		}

		plan, err := tf.ShowPlanFile(ctx, planPath)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, WillDestroy(plan, "random_pet"), true)
	})

	t.Run("return false when a resource is not scheduled for deletion", func(t *testing.T) {
		ctx := context.Background()

		workDir, err := ioutil.TempDir("", "no-deletion")
		if err != nil {
			t.Fatal(err)
		}

		defer os.RemoveAll(workDir)

		tf, err := NewTerraformExec(ctx, workDir, "1.0.3")
		if err != nil {
			t.Fatal(err)
		}

		b := []byte(`
	resource "random_pet" "first" {}
	resource "random_pet" "second" {}
	`)

		if err = ioutil.WriteFile(path.Join(workDir, "main.tf"), b, 0644); err != nil {
			t.Fatal(err)
		}

		if err = tf.Init(ctx); err != nil {
			t.Fatal(err)
		}

		if err = tf.Apply(ctx); err != nil {
			t.Fatal(err)
		}

		b = []byte(`
	resource "random_pet" "first" {}
	resource "random_pet" "second" {}
	resource "random_pet" "third" {}
	`)
		if err = ioutil.WriteFile(path.Join(workDir, "main.tf"), b, 0644); err != nil {
			t.Fatal(err)
		}

		planPath := "plan.txt"

		if _, err = tf.Plan(ctx, tfexec.Out(planPath)); err != nil {
			t.Fatal(err)
		}

		plan, err := tf.ShowPlanFile(ctx, planPath)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, WillDestroy(plan, "random_pet"), false)
	})

	t.Run("return false when a non targetted resource is scheduled for deletion", func(t *testing.T) {
		ctx := context.Background()

		workDir, err := ioutil.TempDir("", "no-deletion")
		if err != nil {
			t.Fatal(err)
		}

		defer os.RemoveAll(workDir)

		tf, err := NewTerraformExec(ctx, workDir, "1.0.3")
		if err != nil {
			t.Fatal(err)
		}

		b := []byte(`
resource "random_pet" "pet" {}
resource "random_id" "id" {
	byte_length = 8
}
`)

		if err = ioutil.WriteFile(path.Join(workDir, "main.tf"), b, 0644); err != nil {
			t.Fatal(err)
		}

		if err = tf.Init(ctx); err != nil {
			t.Fatal(err)
		}

		if err = tf.Apply(ctx); err != nil {
			t.Fatal(err)
		}

		b = []byte(`
resource "random_pet" "pet" {}
`)
		if err = ioutil.WriteFile(path.Join(workDir, "main.tf"), b, 0644); err != nil {
			t.Fatal(err)
		}

		planPath := "plan.txt"

		if _, err = tf.Plan(ctx, tfexec.Out(planPath)); err != nil {
			t.Fatal(err)
		}

		plan, err := tf.ShowPlanFile(ctx, planPath)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, WillDestroy(plan, "random_pet"), false)
	})
}

func TestFindWorkspace(t *testing.T) {
	t.Run("should find a workspace", func(t *testing.T) {
		workspaces := []*Workspace{
			{Name: "foo-staging", Workspace: "staging"},
			{Name: "foo-production", Workspace: "production"},
		}
		assert.Equal(t, FindWorkspace(workspaces, "staging"), workspaces[0])
	})

	t.Run("should not find a workspace", func(t *testing.T) {
		assert.Equal(t, FindWorkspace([]*Workspace{
			{Name: "foo-staging", Workspace: "staging"},
			{Name: "foo-production", Workspace: "production"},
		}, "foo"), (*Workspace)(nil))
	})
}

func TestParseWorkspaces(t *testing.T) {
	t.Run("single workspace", func(t *testing.T) {
		workspaces, err := ParseWorkspaces([]string{}, "foo")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(workspaces), 1)
		assert.Equal(t, workspaces[0], &Workspace{
			Name:      "foo",
			Workspace: "default",
		})
	})

	t.Run("Multiple workspaces", func(t *testing.T) {
		workspaces, err := ParseWorkspaces([]string{"staging", "production"}, "foo")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(workspaces), 2)
		assert.Equal(t, workspaces[0], &Workspace{
			Name:      "foo-staging",
			Workspace: "staging",
		})
		assert.Equal(t, workspaces[1], &Workspace{
			Name:      "foo-production",
			Workspace: "production",
		})
	})
}
