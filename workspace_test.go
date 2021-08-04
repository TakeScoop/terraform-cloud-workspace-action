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
	"gotest.tools/v3/assert"
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

	t.Run("render a full JSON workspace configuration", func(t *testing.T) {
		b, err := json.MarshalIndent(WorkspaceConfig{
			Terraform: WorkspaceTerraform{
				Backend: WorkspaceBackend{
					S3: &S3BackendConfig{},
				},
			},
			Variables: map[string]WorkspaceVariable{
				"workspace_names": {
					Type: "set(string)",
				},
				"variables": {
					Type: "set(map(string))",
				},
			},
			Resources: map[string]map[string]interface{}{
				"tfe_workspace": {
					"workspace": WorkspaceWorkspaceResource{
						ForEach:          "${var.workspace_names}",
						Name:             "${each.value}",
						Organization:     "org",
						AutoApply:        boolPtr(false),
						TerraformVersion: "1.0.0",
						VCSRepo: &WorkspaceVCSBlock{
							OauthTokenID:      "12345",
							Identifier:        "org/repo",
							IngressSubmodules: true,
						},
					},
				},
				"tfe_variable": {
					"variables": WorkspaceVariableResource{
						ForEach:     "${{ for k, v in var.variables : \"${v.workspace_name}-${v.key}\" => v }}",
						Description: "${each.value.description}",
						Key:         "${each.value.key}",
						Value:       "${each.value.value}",
						Category:    "${each.value.category}",
						WorkspaceID: "${tfe_workspace.workspace[each.value.workspace_name].id}",
					},
				},
			},
		}, "", "\t")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, string(b), `{
	"terraform": {
		"backend": {
			"s3": {}
		}
	},
	"variable": {
		"variables": {
			"type": "set(map(string))"
		},
		"workspace_names": {
			"type": "set(string)"
		}
	},
	"resource": {
		"tfe_variable": {
			"variables": {
				"for_each": "${{ for k, v in var.variables : \"${v.workspace_name}-${v.key}\" =\u003e v }}",
				"key": "${each.value.key}",
				"value": "${each.value.value}",
				"description": "${each.value.description}",
				"category": "${each.value.category}",
				"workspace_id": "${tfe_workspace.workspace[each.value.workspace_name].id}"
			}
		},
		"tfe_workspace": {
			"workspace": {
				"for_each": "${var.workspace_names}",
				"auto_apply": false,
				"name": "${each.value}",
				"organization": "org",
				"terraform_version": "1.0.0",
				"vcs_repo": {
					"oauth_token_id": "12345",
					"identifier": "org/repo",
					"ingress_submodules": true
				}
			}
		}
	}
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
		assert.ErrorContains(t, err, "VCS repository must be passed")
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
		assert.DeepEqual(t, ws.RemoteStateConsumerIDs, []string{"123", "456", "789"})
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
		assert.DeepEqual(t, ws.RemoteStateConsumerIDs, []string{})
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

	assert.DeepEqual(t, wsConfig.Data["tfe_team"]["Writers"], TeamDataResource{
		Name:         "Writers",
		Organization: "org",
	})
	assert.DeepEqual(t, wsConfig.Data["tfe_team"]["Readers"], TeamDataResource{
		Name:         "Readers",
		Organization: "org",
	})

	assert.DeepEqual(t,
		wsConfig.Resources["tfe_team_access"]["workspace-Writers"],
		&WorkspaceTeamAccessResource{
			TeamID:      "${data.tfe_team.Writers.id}",
			WorkspaceID: "${tfe_workspace.workspace[\"workspace\"].id}",
			Access:      "write",
		},
	)
	assert.DeepEqual(t,
		wsConfig.Resources["tfe_team_access"]["workspace-Readers"],
		&WorkspaceTeamAccessResource{
			TeamID:      "${data.tfe_team.Readers.id}",
			WorkspaceID: "${tfe_workspace.workspace[\"workspace\"].id}",
			Access:      "read",
		},
	)
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
			TerraformBackendConfig: &WorkspaceTerraform{
				Backend: WorkspaceBackend{
					Local: &LocalBackendConfig{},
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
			TerraformBackendConfig: &WorkspaceTerraform{
				Backend: WorkspaceBackend{
					Local: &LocalBackendConfig{},
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
			TerraformBackendConfig: &WorkspaceTerraform{
				Backend: WorkspaceBackend{
					Local: &LocalBackendConfig{},
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
			TerraformBackendConfig: &WorkspaceTerraform{
				Backend: WorkspaceBackend{
					Local: &LocalBackendConfig{},
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
