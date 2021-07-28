package main

import (
	"context"
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

	t.Run("get client by type", func(t *testing.T) {
		tokenID, err := GetVCSTokenIDByClientType(ctx, client, "org", "github")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, tokenID, "ot-678910")
	})
}
