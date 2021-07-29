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

func TestGetVCSTokenIDByClientType(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	defer server.Close()

	mux.HandleFunc("/api/v2/organizations/org/oauth-clients", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		_, err := fmt.Fprint(w, `
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
		`)
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
	t.Run("render a full JSON workspace configuration", func(t *testing.T) {
		b, err := json.MarshalIndent(WorkspaceResource{
			ForEach:          "${var.workspace_names}",
			Name:             "${each.value}",
			Organization:     "${var.organization}",
			AutoApply:        true,
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
		b, err := json.MarshalIndent(WorkspaceConfig{
			Terraform: WorkspaceTerraform{
				Backend: WorkspaceBackend{
					S3: WorkspaceBackend{},
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
			Resources: map[string]map[string]WorkspaceResource{
				"tfe_workspace": {
					"workspace": {
						ForEach:          "${var.workspace_names}",
						Name:             "${each.value}",
						Organization:     "${var.organization}",
						AutoApply:        true,
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
