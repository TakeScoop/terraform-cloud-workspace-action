package action

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

type TestTFExec struct {
	State      *tfjson.State
	ImportArgs *ImportArgs
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
	tf.ImportArgs = &ImportArgs{
		Address: address,
		ID:      ID,
		Opts:    opts,
	}

	return nil
}

func TestImportWorkspace(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	defer server.Close()

	mux.HandleFunc("/api/v2/organizations/org/workspaces/ws", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		_, err := fmt.Fprint(w, `{"data":{"id":"ws-abc123","type":"workspaces","attributes":{"allow-destroy-plan":false,"auto-apply":false,"auto-destroy-at":null,"created-at":"2021-08-26T04:43:54.557Z","environment":"default","locked":false,"name":"ws","queue-all-runs":true,"speculative-enabled":true,"structured-run-output-enabled":true,"terraform-version":"1.0.5","working-directory":"","global-remote-state":false,"updated-at":"2021-08-26T04:43:54.557Z","resource-count":0,"apply-duration-average":null,"plan-duration-average":null,"policy-check-failures":null,"run-failures":null,"workspace-kpis-runs-count":null,"latest-change-at":"2021-08-26T04:43:54.557Z","operations":true,"execution-mode":"remote","vcs-repo":null,"vcs-repo-identifier":null,"permissions":{"can-update":true,"can-destroy":true,"can-queue-destroy":true,"can-queue-run":true,"can-queue-apply":true,"can-read-state-versions":true,"can-create-state-versions":true,"can-read-variable":true,"can-update-variable":true,"can-lock":true,"can-unlock":true,"can-force-unlock":true,"can-read-settings":true,"can-manage-tags":true},"actions":{"is-destroyable":false},"description":"","file-triggers-enabled":true,"trigger-prefixes":[],"source":"tfe-api","source-name":null,"source-url":null,"tag-names":[]},"relationships":{"organization":{"data":{"id":"org","type":"organizations"}},"current-run":{"data":null},"latest-run":{"data":null},"outputs":{"data":[]},"remote-state-consumers":{"links":{"related":"/api/v2/workspaces/ws-abc123/relationships/remote-state-consumers"}},"current-state-version":{"data":null},"current-configuration-version":{"data":null},"agent-pool":{"data":null},"readme":{"data":null}},"links":{"self":"/api/v2/organizations/org/workspaces/ws"}}}`)
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

	t.Run("skip importing if it already exists in state", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{
				Values: &tfjson.StateValues{
					RootModule: &tfjson.StateModule{
						Resources: []*tfjson.StateResource{
							{Address: "tfe_workspace.workspace[\"ws\"]"},
						},
					},
				},
			},
		}

		if err := ImportWorkspace(ctx, &tf, client, "ws", "org"); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, tf.ImportArgs, (*ImportArgs)(nil))
	})

	t.Run("import the workspace if it does not exist in state", func(t *testing.T) {
		tf := TestTFExec{
			State: &tfjson.State{},
		}

		if err := ImportWorkspace(ctx, &tf, client, "ws", "org"); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, tf.ImportArgs, &ImportArgs{
			Address: "tfe_workspace.workspace[\"ws\"]",
			ID:      "ws-abc123",
			Opts:    ([]tfexec.ImportOption)(nil),
		})
	})
}
