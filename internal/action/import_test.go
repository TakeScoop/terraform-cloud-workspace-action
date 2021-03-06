package action

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

type TestTFExec struct {
	State      *tfjson.State
	ImportArgs []*ImportArgs
}

type ImportArgs struct {
	Address string
	ID      string
	Opts    []tfexec.ImportOption
}

func (tf TestTFExec) Show(ctx context.Context, opts ...tfexec.ShowOption) (*tfjson.State, error) {
	return tf.State, nil
}

func (tf *TestTFExec) Import(ctx context.Context, address string, ID string, opts ...tfexec.ImportOption) error {
	tf.ImportArgs = append(tf.ImportArgs, &ImportArgs{
		Address: address,
		ID:      ID,
		Opts:    opts,
	})

	return nil
}

func strPtr(s string) *string {
	return &s
}

func TestImportWorkspace(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
	})

	client := newTestTFClient(t, server.URL)

	t.Run("skip importing if it already exists in state", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{
				Values: &tfjson.StateValues{
					RootModule: &tfjson.StateModule{
						Resources: []*tfjson.StateResource{
							{Address: "tfe_workspace.workspace[\"default\"]"},
						},
					},
				},
			},
		}

		if err := ImportWorkspace(ctx, &tf, client, &Workspace{Name: "ws", Workspace: "default", ID: strPtr("ws-abc123")}, "org"); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(tf.ImportArgs), 0)
	})

	t.Run("import the workspace if it does not exist in state", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{},
		}

		if err := ImportWorkspace(ctx, &tf, client, &Workspace{Name: "ws", Workspace: "default", ID: strPtr("ws-abc123")}, "org"); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(tf.ImportArgs), 1)
		assert.Equal(t, tf.ImportArgs[0], &ImportArgs{
			Address: "tfe_workspace.workspace[\"default\"]",
			ID:      "ws-abc123",
			Opts:    ([]tfexec.ImportOption)(nil),
		})
	})

	t.Run("skip importing the workspace if the workspace was not set with an ID", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{},
		}

		if err := ImportWorkspace(ctx, &tf, client, &Workspace{Name: "ws", Workspace: "default", ID: nil}, "org"); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(tf.ImportArgs), 0)
	})
}

func TestImportVariable(t *testing.T) {
	ctx := context.Background()

	t.Run("import a variable", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{
				Values: &tfjson.StateValues{
					RootModule: &tfjson.StateModule{
						Resources: []*tfjson.StateResource{
							{Address: "tfe_workspace.workspace[\"default\"]"},
						},
					},
				},
			},
		}

		if err := ImportVariable(ctx, &tf, &tfe.Variable{
			Key: "foo",
			ID:  "var-abc123",
		}, &Workspace{Name: "ws", Workspace: "default", ID: strPtr("ws-abc123")}, "org"); err != nil {
			t.Fatal(err)
		}

		assert.Len(t, tf.ImportArgs, 1)
		assert.Equal(t, tf.ImportArgs[0], &ImportArgs{
			Address: "tfe_variable.default-foo",
			ID:      "org/ws/var-abc123",
			Opts:    ([]tfexec.ImportOption)(nil),
		})
	})

	t.Run("skip importing a variable if the workspace is not set with an ID", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{},
		}

		if err := ImportVariable(ctx, &tf, &tfe.Variable{Key: "foo", ID: "var-abc123"}, &Workspace{Name: "ws", ID: nil}, "org"); err != nil {
			t.Fatal(err)
		}

		assert.Len(t, tf.ImportArgs, 0)
	})
}

func TestImportTeamAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("import a team access resource", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{
				Values: &tfjson.StateValues{
					RootModule: &tfjson.StateModule{
						Resources: []*tfjson.StateResource{
							{Address: "tfe_workspace.workspace[\"default\"]"},
						},
					},
				},
			},
		}

		if err := ImportTeamAccess(ctx, &tf, &tfe.TeamAccess{
			ID: "tws-abc123",
			Team: &tfe.Team{
				ID: "team-abc123",
			},
		}, &Workspace{Name: "ws", Workspace: "default", ID: strPtr("ws-abc123")}, "org"); err != nil {
			t.Fatal(err)
		}

		assert.Len(t, tf.ImportArgs, 1)
		assert.Equal(t, &ImportArgs{
			Address: "tfe_team_access.teams[\"default-team-abc123\"]",
			ID:      "org/ws/tws-abc123",
			Opts:    ([]tfexec.ImportOption)(nil),
		}, tf.ImportArgs[0])
	})

	t.Run("skip import if the workspace is not set with an ID", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{},
		}

		if err := ImportTeamAccess(ctx, &tf, &tfe.TeamAccess{
			ID:   "tws-abc123",
			Team: &tfe.Team{ID: "team-abc123"},
		}, &Workspace{Name: "ws", Workspace: "default", ID: nil}, "org"); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(tf.ImportArgs), 0)
	})

	t.Run("skip import if the team access already exists in state", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{
				Values: &tfjson.StateValues{
					RootModule: &tfjson.StateModule{
						Resources: []*tfjson.StateResource{
							{Address: "tfe_workspace.workspace[\"default\"]"},
							{Address: "tfe_team_access.teams[\"default-team-abc123\"]"},
						},
					},
				},
			},
		}

		if err := ImportTeamAccess(ctx, &tf, &tfe.TeamAccess{
			ID:   "tws-abc123",
			Team: &tfe.Team{ID: "team-abc123"},
		}, &Workspace{Name: "ws", Workspace: "default", ID: tfe.String("ws-abc123")}, "org"); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(tf.ImportArgs), 0)
	})
}

func TestImportRelatedRunTriggers(t *testing.T) {
	ctx := context.Background()

	t.Run("import a team access resource", func(t *testing.T) {
		mux := http.NewServeMux()
		server := httptest.NewServer(mux)

		t.Cleanup(func() {
			server.Close()
		})

		mux.HandleFunc("/api/v2/workspaces/ws-abc123/run-triggers", testServerResHandler(t, 200, runTriggerAPIResponse))

		client := newTestTFClient(t, server.URL)

		tf := TestTFExec{
			State: &tfjson.State{},
		}
		workspace := newTestWorkspace()

		triggers, err := FetchInboundRunTriggers(ctx, client, *workspace.ID)
		assert.NoError(t, err)

		err = ImportRunTriggers(ctx, &tf, triggers, client, workspace)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(tf.ImportArgs), 1)
		assert.Equal(t, tf.ImportArgs[0], &ImportArgs{
			Address: "tfe_run_trigger.trigger[\"default-ws-def456\"]",
			ID:      "rt-abc123",
			Opts:    ([]tfexec.ImportOption)(nil),
		})
	})

	t.Run("no import if no related run triggers", func(t *testing.T) {
		mux := http.NewServeMux()
		server := httptest.NewServer(mux)

		t.Cleanup(func() {
			server.Close()
		})

		mux.HandleFunc("/api/v2/workspaces/ws-abc123/run-triggers", testServerResHandler(t, 200, `{"data": []}`))

		client := newTestTFClient(t, server.URL)

		tf := TestTFExec{
			State: &tfjson.State{},
		}

		workspace := newTestWorkspace()

		triggers, err := FetchInboundRunTriggers(ctx, client, *workspace.ID)
		assert.NoError(t, err)

		err = ImportRunTriggers(ctx, &tf, triggers, client, workspace)
		assert.NoError(t, err)

		assert.Len(t, tf.ImportArgs, 0)
	})
}

var runTriggerAPIResponse string = `{
  "data": [
    {
      "id": "rt-abc123",
      "type": "run-triggers",
      "attributes": {
        "workspace-name": "ws",
        "sourceable-name": "ws-sourceable",
        "created-at": "2021-09-16T05:36:15.784Z"
      },
      "relationships": {
        "workspace": {
          "data": {
            "id": "ws-abc123",
            "type": "workspaces"
          }
        },
        "sourceable": {
          "data": {
            "id": "ws-def456",
            "type": "workspaces"
          }
        }
      },
      "links": {
        "self": "/api/v2/run-triggers/rt-abc123"
      }
    }
  ],
  "included": [
    {
      "id": "ws-abc123",
      "type": "workspaces",
      "attributes": {
        "allow-destroy-plan": true,
        "auto-apply": true,
        "auto-destroy-at": null,
        "created-at": "2021-09-09T03:12:02.113Z",
        "environment": "default",
        "locked": false,
        "name": "deployments-test-staging",
        "queue-all-runs": true,
        "speculative-enabled": true,
        "structured-run-output-enabled": true,
        "terraform-version": "1.0.5",
        "working-directory": "",
        "global-remote-state": false,
        "updated-at": "2021-09-10T19:37:18.761Z",
        "resource-count": 0,
        "apply-duration-average": null,
        "plan-duration-average": null,
        "policy-check-failures": null,
        "run-failures": null,
        "workspace-kpis-runs-count": null,
        "latest-change-at": "2021-09-09T03:12:02.113Z",
        "operations": true,
        "execution-mode": "remote",
        "vcs-repo": {
          "branch": "",
          "ingress-submodules": false,
          "identifier": "org/ws",
          "display-identifier": "org/ws",
          "oauth-token-id": "ot-abc123",
          "webhook-url": "https://app.terraform.io/webhooks/vcs/00000000-2a86-4655-a7bc-d48ecb2e09f6",
          "repository-http-url": "https://github.com/org/ws",
          "service-provider": "github"
        },
        "vcs-repo-identifier": "org/ws",
        "permissions": {
          "can-update": true,
          "can-destroy": true,
          "can-queue-destroy": true,
          "can-queue-run": true,
          "can-queue-apply": true,
          "can-read-state-versions": true,
          "can-create-state-versions": true,
          "can-read-variable": true,
          "can-update-variable": true,
          "can-lock": true,
          "can-unlock": true,
          "can-force-unlock": true,
          "can-read-settings": true,
          "can-manage-tags": true
        },
        "actions": {
          "is-destroyable": true
        },
        "description": "test workspace foo",
        "file-triggers-enabled": true,
        "trigger-prefixes": [],
        "source": "tfe-api",
        "source-name": null,
        "source-url": null,
        "tag-names": []
      },
      "relationships": {
        "organization": {
          "data": {
            "id": "org",
            "type": "organizations"
          }
        },
        "current-run": {
          "data": {
            "id": "run-abc123",
            "type": "runs"
          },
          "links": {
            "related": "/api/v2/runs/run-abc123"
          }
        },
        "latest-run": {
          "data": {
            "id": "run-abc123",
            "type": "runs"
          },
          "links": {
            "related": "/api/v2/runs/run-abc123"
          }
        },
        "outputs": {
          "data": []
        },
        "remote-state-consumers": {
          "links": {
            "related": "/api/v2/workspaces/ws-abc123/relationships/remote-state-consumers"
          }
        },
        "current-state-version": {
          "data": null
        },
        "current-configuration-version": {
          "data": {
            "id": "cv-abc123",
            "type": "configuration-versions"
          },
          "links": {
            "related": "/api/v2/configuration-versions/cv-abc123"
          }
        },
        "agent-pool": {
          "data": null
        },
        "readme": {
          "data": {
            "id": "254797",
            "type": "workspace-readme"
          }
        }
      },
      "links": {
        "self": "/api/v2/organizations/org/workspaces/ws"
      }
    }
  ],
  "links": {
    "self": "https://app.terraform.io/api/v2/workspaces/ws-abc123/run-triggers?filter%5Brun-trigger%5D%5Btype%5D=inbound\u0026include=sourceable\u0026page%5Bnumber%5D=1\u0026page%5Bsize%5D=5\u0026workspace_id=ws-ijhdfBHEMWagkig3",
    "first": "https://app.terraform.io/api/v2/workspaces/ws-abc123/run-triggers?filter%5Brun-trigger%5D%5Btype%5D=inbound\u0026include=sourceable\u0026page%5Bnumber%5D=1\u0026page%5Bsize%5D=5\u0026workspace_id=ws-ijhdfBHEMWagkig3",
    "prev": null,
    "next": null,
    "last": "https://app.terraform.io/api/v2/workspaces/ws-abc123/run-triggers?filter%5Brun-trigger%5D%5Btype%5D=inbound\u0026include=sourceable\u0026page%5Bnumber%5D=1\u0026page%5Bsize%5D=5\u0026workspace_id=ws-ijhdfBHEMWagkig3"
  },
  "meta": {
    "pagination": {
      "current-page": 1,
      "page-size": 5,
      "prev-page": null,
      "next-page": null,
      "total-pages": 1,
      "total-count": 1
    }
  }
}`
