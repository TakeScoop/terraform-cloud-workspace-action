package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
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

	mux.HandleFunc("/api/v2/organizations/org/oauth-clients", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		_, err := fmt.Fprint(w, basicOauthClientResponse)
		if err != nil {
			t.Fatal(err)
		}
	})

	client, err := tfe.NewClient(&tfe.Config{
		Address: server.URL,
		Token:   "12345",
	})
	if err != nil {
		t.Fatal(err)
	}

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
		b, err := json.MarshalIndent(WorkspaceWorkspaceResource{
			ForEach:          "${var.workspace_names}",
			Name:             "${each.value}",
			Organization:     "${var.organization}",
			AutoApply:        boolPtr(true),
			TerraformVersion: "${var.terraform_version}",
			VCSRepo:          nil,
		}, "", "\t")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, string(b), `{
	"for_each": "${var.workspace_names}",
	"auto_apply": true,
	"name": "${each.value}",
	"organization": "${var.organization}",
	"terraform_version": "${var.terraform_version}"
}`)
	})
}

func TestNewWorkspaceResource(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	defer server.Close()

	mux.HandleFunc("/api/v2/organizations/org/oauth-clients", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		_, err := fmt.Fprint(w, basicOauthClientResponse)
		if err != nil {
			t.Fatal(err)
		}
	})

	client, err := tfe.NewClient(&tfe.Config{
		Address: server.URL,
		Token:   "12345",
	})
	if err != nil {
		t.Fatal(err)
	}

	type BoolTest struct {
		AutoApply *bool `json:"auto_apply,omitempty"`
	}

	t.Run("should render a basic workspace without unprovided values", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
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
	"for_each": "${var.workspace_names}",
	"name": "${each.value}",
	"organization": "org"
}`)
	})

	t.Run("should set boolean value to false if passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
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
		_, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
			Organization: "org",
			VCSType:      "github",
		})
		assert.EqualError(t, err, "VCS repository must be passed if VCS type or a VCS token ID is passed")
	})

	t.Run("use VCSTokenID directly when passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, &WorkspaceResourceOptions{
			Organization:      "org",
			GlobalRemoteState: boolPtr(false),
		})
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, *ws.GlobalRemoteState, false)
		assert.Equal(t, ws.RemoteStateConsumerIDs, []string{})
	})
}

func TestAddRemoteStates(t *testing.T) {
	wsConfig := WorkspaceConfig{
		Data: map[string]map[string]interface{}{},
	}

	foo := RemoteState{Backend: "S3", Config: RemoteStateBackendConfig{}}
	bar := RemoteState{Backend: "S3", Config: RemoteStateBackendConfig{}}
	wsConfig.AddRemoteStates(map[string]RemoteState{
		"foo": foo,
		"bar": bar,
	})
	assert.Equal(t, wsConfig.Data["terraform_remote_state"]["foo"], foo)
	assert.Equal(t, wsConfig.Data["terraform_remote_state"]["bar"], bar)
}

func TestAddTeamAccess(t *testing.T) {
	wsConfig := WorkspaceConfig{
		Data:      map[string]map[string]interface{}{},
		Resources: map[string]map[string]interface{}{},
	}

	wsConfig.AddTeamAccess([]TeamAccess{
		{TeamName: "Writers", Access: "write", WorkspaceName: "workspace"},
		{TeamName: "Readers", Access: "read", WorkspaceName: "workspace"},
	}, "org")

	assert.Equal(t, wsConfig.Data["tfe_team"]["Writers"], TeamDataResource{
		Name:         "Writers",
		Organization: "org",
	})
	assert.Equal(t, wsConfig.Data["tfe_team"]["Readers"], TeamDataResource{
		Name:         "Readers",
		Organization: "org",
	})

	assert.Equal(t,
		wsConfig.Resources["tfe_team_access"]["workspace-Writers"],
		&WorkspaceTeamAccessResource{
			TeamID:      "${data.tfe_team.Writers.id}",
			WorkspaceID: "${tfe_workspace.workspace[\"workspace\"].id}",
			Access:      "write",
		},
	)
	assert.Equal(t,
		wsConfig.Resources["tfe_team_access"]["workspace-Readers"],
		&WorkspaceTeamAccessResource{
			TeamID:      "${data.tfe_team.Readers.id}",
			WorkspaceID: "${tfe_workspace.workspace[\"workspace\"].id}",
			Access:      "read",
		},
	)
}

func TestAddProviders(t *testing.T) {
	wsConfig := WorkspaceConfig{
		Data:      map[string]map[string]interface{}{},
		Resources: map[string]map[string]interface{}{},
	}

	wsConfig.AddProviders([]Provider{
		{Name: "tfe", Version: "0.25.0", Source: "hashicorp/tfe", Config: TFEProvider{Hostname: "app.terraform.io"}},
	})

	assert.Equal(t, wsConfig.Providers["tfe"].(TFEProvider).Hostname, "app.terraform.io")
	assert.Equal(t, wsConfig.Terraform.RequiredProviders["tfe"].Source, "hashicorp/tfe")
	assert.Equal(t, wsConfig.Terraform.RequiredProviders["tfe"].Version, "0.25.0")
}

func RunValidate(ctx context.Context, name string, tfexecPath string, wsConfig *WorkspaceConfig) (*tfjson.ValidateOutput, error) {
	b, err := json.MarshalIndent(wsConfig, "", "\t")
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

	mux.HandleFunc("/api/v2/organizations/org/oauth-clients", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		_, err := fmt.Fprint(w, basicOauthClientResponse)
		if err != nil {
			t.Fatal(err)
		}
	})

	client, err := tfe.NewClient(&tfe.Config{
		Address: server.URL,
		Token:   "12345",
	})
	if err != nil {
		t.Fatal(err)
	}

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
		wsConfig, err := NewWorkspaceConfig(ctx, client, &NewWorkspaceConfigOptions{
			Backend: &WorkspaceBackend{
				Local: &LocalBackendConfig{},
			},
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
			WorkspaceVariables: map[string]WorkspaceVariable{
				"workspace_names": {
					Type: "set(string)",
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

	t.Run("validate workspace with passed providers", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client, &NewWorkspaceConfigOptions{
			Backend: &WorkspaceBackend{
				Local: &LocalBackendConfig{},
			},
			Providers: []Provider{
				{
					Name:    "tfe",
					Version: "0.25.0",
					Source:  "hashicorp/tfe",
					Config: TFEProvider{
						Hostname: "app.terraform.io",
					},
				},
			},
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
			WorkspaceVariables: map[string]WorkspaceVariable{
				"workspace_names": {
					Type: "set(string)",
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

	t.Run("validate workspace with remote states", func(t *testing.T) {
		wsConfig, err := NewWorkspaceConfig(ctx, client, &NewWorkspaceConfigOptions{
			Backend: &WorkspaceBackend{
				Local: &LocalBackendConfig{},
			},
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
			WorkspaceVariables: map[string]WorkspaceVariable{
				"workspace_names": {
					Type: "set(string)",
				},
			},
			RemoteStates: map[string]RemoteState{
				"foo": {
					Backend: "s3",
					Config: RemoteStateBackendConfig{
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
		wsConfig, err := NewWorkspaceConfig(ctx, client, &NewWorkspaceConfigOptions{
			Backend: &WorkspaceBackend{
				Local: &LocalBackendConfig{},
			},
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
			WorkspaceVariables: map[string]WorkspaceVariable{
				"workspace_names": {
					Type: "set(string)",
				},
			},
			TeamAccess: []TeamAccess{
				{TeamName: "Readers", WorkspaceName: name, Access: "read"},
				{TeamName: "Writers", WorkspaceName: name, Permissions: &TeamAccessPermissions{
					Runs:             "read",
					Variables:        "read",
					StateVersions:    "read",
					SentinelMocks:    "none",
					WorkspaceLocking: true,
				}},
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
		wsConfig, err := NewWorkspaceConfig(ctx, client, &NewWorkspaceConfigOptions{
			Backend: &WorkspaceBackend{
				Local: &LocalBackendConfig{},
			},
			WorkspaceResourceOptions: &WorkspaceResourceOptions{
				Organization: "org",
			},
			WorkspaceVariables: map[string]WorkspaceVariable{
				"workspace_names": {
					Type: "set(string)",
				},
			},
			Variables: []Variable{
				{Key: "foo", Value: "bar", Category: "env", WorkspaceName: name},
				{Key: "baz", Value: "woz", Category: "env", WorkspaceName: name},
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
			t.Fatal()
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
			t.Fatal()
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
}
