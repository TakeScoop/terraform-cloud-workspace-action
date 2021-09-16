package action

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

func TestRunTriggerToResource(t *testing.T) {
	rt := RunTrigger{
		SourceID:  "ws-def456",
		Workspace: newTestWorkspace(),
	}

	assert.Equal(t, &tfeprovider.RunTrigger{
		SourceableID: "ws-def456",
		WorkspaceID:  "${tfe_workspace.workspace[\"default\"].id}",
	}, rt.ToResource())
}

func TestMergeRunTriggers(t *testing.T) {
	t.Run("sourceID is used", func(t *testing.T) {
		inputs := RunTriggerInputs{
			{SourceID: "ws-def456"},
		}
		wsInputs := map[string]RunTriggerInputs{}
		workspaces := newTestSingleWorkspaceList()

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces, "org")
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{SourceID: "ws-def456", Workspace: workspaces[0]},
		}, triggers)
	})

	t.Run("sourceName is used for a workspace to be created", func(t *testing.T) {
		workspaces := newTestSingleWorkspaceList()
		inputs := RunTriggerInputs{
			{SourceName: workspaces[0].Name},
		}
		wsInputs := map[string]RunTriggerInputs{}

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces, "org")
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{
				SourceID:  "${tfe_workspace.workspace[\"default\"].id}",
				Workspace: workspaces[0],
				DataRef:   (map[string]tfeprovider.DataWorkspace)(nil),
			}}, triggers)
	})

	t.Run("sourceName is used for a workspace not otherwise referenced in the workspace config", func(t *testing.T) {
		workspaces := newTestSingleWorkspaceList()
		inputs := RunTriggerInputs{
			{SourceName: "foo"},
		}
		wsInputs := map[string]RunTriggerInputs{}

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces, "org")
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{
				SourceID:  "${data.tfe_workspace.run_trigger_workspaces[\"foo\"].id}",
				Workspace: workspaces[0],
				DataRef: map[string]tfeprovider.DataWorkspace{
					"foo": {
						Name:         "foo",
						Organization: "org",
					},
				},
			}}, triggers)
	})

	t.Run("single workspaces", func(t *testing.T) {
		inputs := RunTriggerInputs{
			{SourceID: "ws-def456"},
			{SourceID: "ws-ghi789"},
		}
		wsInputs := map[string]RunTriggerInputs{}
		workspaces := newTestSingleWorkspaceList()

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces, "org")
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{SourceID: "ws-def456", Workspace: workspaces[0]},
			{SourceID: "ws-ghi789", Workspace: workspaces[0]},
		}, triggers)
	})

	t.Run("multi workspaces", func(t *testing.T) {
		inputs := RunTriggerInputs{
			{SourceID: "ws-def456"},
			{SourceID: "ws-ghi789"},
		}
		wsInputs := map[string]RunTriggerInputs{}
		workspaces := newTestMultiWorkspaceList()

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces, "org")
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{SourceID: "ws-def456", Workspace: workspaces[0]},
			{SourceID: "ws-def456", Workspace: workspaces[1]},
			{SourceID: "ws-ghi789", Workspace: workspaces[0]},
			{SourceID: "ws-ghi789", Workspace: workspaces[1]},
		}, triggers)
	})

	t.Run("empty run triggers list", func(t *testing.T) {
		inputs := RunTriggerInputs{}
		wsInputs := map[string]RunTriggerInputs{}

		workspaces := newTestMultiWorkspaceList()

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces, "org")
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{}, triggers)
	})

	t.Run("workspaced triggers", func(t *testing.T) {
		inputs := RunTriggerInputs{}
		workspaces := newTestMultiWorkspaceList()
		wsInputs := map[string]RunTriggerInputs{
			workspaces[0].Workspace: {{SourceID: "ws-def456"}},
		}

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces, "org")
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{SourceID: "ws-def456", Workspace: workspaces[0]},
		}, triggers)
	})

	t.Run("workspaced triggers with sourceName", func(t *testing.T) {
		inputs := RunTriggerInputs{}
		workspaces := newTestMultiWorkspaceList()
		wsInputs := map[string]RunTriggerInputs{
			workspaces[0].Workspace: {{SourceName: "foo"}},
		}

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces, "org")
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{
				SourceID:  "${data.tfe_workspace.run_trigger_workspaces[\"foo\"].id}",
				Workspace: workspaces[0],
				DataRef: map[string]tfeprovider.DataWorkspace{
					"foo": {
						Name:         "foo",
						Organization: "org",
					},
				},
			},
		}, triggers)
	})
}

func TestAppendRunTriggers(t *testing.T) {
	t.Run("triggers by ID", func(t *testing.T) {
		module := NewModule()

		workspace := newTestWorkspace()

		AppendRunTriggers(module, RunTriggers{
			{SourceID: "ws-abc123", Workspace: workspace},
		})

		assert.Equal(t, tfeprovider.RunTrigger{
			ForEach: map[string]tfeprovider.RunTrigger{
				"default-ws-abc123": {
					SourceableID: "ws-abc123",
					WorkspaceID:  "${tfe_workspace.workspace[\"default\"].id}",
				},
			},
			SourceableID: "${each.value.sourceable_id}",
			WorkspaceID:  "${each.value.workspace_id}",
		}, module.Resources["tfe_run_trigger"]["trigger"])

		assert.Equal(t, map[string]map[string]interface{}{}, module.Data)
	})

	t.Run("triggers with data ref", func(t *testing.T) {
		module := NewModule()

		workspace := newTestWorkspace()

		AppendRunTriggers(module, RunTriggers{
			{
				SourceID:  "${data.tfe_workspace.run_trigger_workspaces[\"foo\"].id}",
				Workspace: workspace,
				DataRef: map[string]tfeprovider.DataWorkspace{
					"foo": {
						Name:         "foo",
						Organization: "org",
					},
				},
			},
		})

		assert.Equal(t, tfeprovider.RunTrigger{
			ForEach: map[string]tfeprovider.RunTrigger{
				"default-${data.tfe_workspace.run_trigger_workspaces[\"foo\"].id}": {
					SourceableID: "${data.tfe_workspace.run_trigger_workspaces[\"foo\"].id}",
					WorkspaceID:  "${tfe_workspace.workspace[\"default\"].id}",
				},
			},
			SourceableID: "${each.value.sourceable_id}",
			WorkspaceID:  "${each.value.workspace_id}",
		}, module.Resources["tfe_run_trigger"]["trigger"])

		assert.Equal(t, map[string]map[string]interface{}{
			"tfe_workspace": {
				"run_trigger_workspaces": tfeprovider.DataWorkspace{
					ForEach: map[string]tfeprovider.DataWorkspace{
						"foo": {
							Name:         "foo",
							Organization: "org",
						},
					},
					Name:         "${each.value.name}",
					Organization: "${each.value.organization}",
				},
			},
		}, module.Data)
	})
}

func TestToRunTriggers(t *testing.T) {
	t.Run("get a list of RunTriggers", func(t *testing.T) {
		ctx := context.Background()

		mux := http.NewServeMux()
		server := httptest.NewServer(mux)

		defer server.Close()

		mux.HandleFunc("/api/v2/workspaces/ws-abc123/run-triggers", testServerResHandler(t, 200, runTriggerAPIResponse))

		client := newTestTFClient(t, server.URL)

		workspace := newTestWorkspace()

		tfeTriggers, err := FetchInboundRunTriggers(ctx, client, *workspace.ID)
		if err != nil {
			t.Fatal(err)
		}

		triggers := ToRunTriggers(tfeTriggers, workspace)

		assert.Len(t, triggers, 1)
		assert.Equal(t, RunTriggers{
			{SourceID: "ws-def456", Workspace: workspace},
		}, triggers)
	})

	t.Run("empty list of RunTriggers when none exist", func(t *testing.T) {
		ctx := context.Background()

		mux := http.NewServeMux()
		server := httptest.NewServer(mux)

		defer server.Close()

		mux.HandleFunc("/api/v2/workspaces/ws-abc123/run-triggers", testServerResHandler(t, 200, `{"data":[]}`))

		client := newTestTFClient(t, server.URL)

		workspace := newTestWorkspace()

		tfeTriggers, err := FetchInboundRunTriggers(ctx, client, *workspace.ID)
		if err != nil {
			t.Fatal(err)
		}

		triggers := ToRunTriggers(tfeTriggers, workspace)

		assert.Len(t, triggers, 0)
	})
}
