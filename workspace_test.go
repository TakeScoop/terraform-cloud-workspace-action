package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"gotest.tools/assert"
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

func stringPtr(s string) *string {
	return &s
}

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
		autoApply := true
		b, err := json.MarshalIndent(WorkspaceWorkspaceResource{
			ForEach:          "${var.workspace_names}",
			Name:             "${each.value}",
			Organization:     "${var.organization}",
			AutoApply:        &autoApply,
			TerraformVersion: "${var.terraform_version}",
			VCSRepo:          nil,
		}, "", "\t")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, string(b), `{
	"for_each": "${var.workspace_names}",
	"name": "${each.value}",
	"organization": "${var.organization}",
	"auto_apply": true,
	"terraform_version": "${var.terraform_version}"
}`)
	})

	t.Run("render a full JSON workspace configuration", func(t *testing.T) {
		autoApply := true
		b, err := json.MarshalIndent(WorkspaceConfig{
			Terraform: WorkspaceTerraform{
				Backend: WorkspaceBackend{
					S3: S3BackendConfig{},
				},
			},
			Variables: map[string]WorkspaceVariable{
				"organization": {
					Type: "string",
				},
				"terraform_version": {
					Type: "string",
				},
				"workspace_names": {
					Type: "set(string)",
				},
			},
			Resources: map[string]map[string]interface{}{
				"tfe_workspace": {
					"workspace": WorkspaceWorkspaceResource{
						ForEach:          "${var.workspace_names}",
						Name:             "${each.value}",
						Organization:     "${var.organization}",
						AutoApply:        &autoApply,
						TerraformVersion: "${var.terraform_version}",
						VCSRepo: &WorkspaceVCSBlock{
							OauthTokenID:      "12345",
							Identifier:        "org/repo",
							IngressSubmodules: true,
						},
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
		"organization": {
			"type": "string"
		},
		"terraform_version": {
			"type": "string"
		},
		"workspace_names": {
			"type": "set(string)"
		}
	},
	"resource": {
		"tfe_workspace": {
			"workspace": {
				"for_each": "${var.workspace_names}",
				"name": "${each.value}",
				"organization": "${var.organization}",
				"auto_apply": true,
				"terraform_version": "${var.terraform_version}",
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
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
			Organization: "org",
			VCSType:      stringPtr("github"),
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
		_, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
			Organization: "org",
			VCSType:      stringPtr("github"),
		})
		assert.ErrorContains(t, err, "vcs_repo must be passed")
	})

	t.Run("use VCSTokenID directly when passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
			Organization: "org",
			VCSTokenID:   stringPtr("TOKEN"),
			VCSType:      stringPtr("github"),
			VCSRepo:      "org/repo",
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, ws.VCSRepo.OauthTokenID, "TOKEN")
	})

	t.Run("add AgentPoolID and ExecutionMode: \"agent\" when AgentPoolID is passed", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
			Organization: "org",
			AgentPoolID:  stringPtr("12345"),
		})
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, ws.AgentPoolID, "12345")
		assert.Equal(t, ws.ExecutionMode, "agent")
	})

	t.Run("add RemoteConsumerIDs and GlobalRemoteState if global_remote_state is false", func(t *testing.T) {
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
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
		ws, err := NewWorkspaceResource(ctx, client, WorkspaceConfigOptions{
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
