package action

import (
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
	t.Run("single workspaces", func(t *testing.T) {
		inputs := RunTriggerInputs{"ws-def456", "ws-ghi789"}
		wsInputs := map[string]RunTriggerInputs{}
		workspaces := newTestSingleWorkspaceList()

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces)
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{SourceID: "ws-def456", Workspace: workspaces[0]},
			{SourceID: "ws-ghi789", Workspace: workspaces[0]},
		}, triggers)
	})

	t.Run("multi workspaces", func(t *testing.T) {
		inputs := RunTriggerInputs{"ws-def456", "ws-ghi789"}
		wsInputs := map[string]RunTriggerInputs{}
		workspaces := newTestMultiWorkspaceList()

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces)
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

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces)
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{}, triggers)
	})

	t.Run("workspaced triggers", func(t *testing.T) {
		inputs := RunTriggerInputs{}
		workspaces := newTestMultiWorkspaceList()
		wsInputs := map[string]RunTriggerInputs{
			workspaces[0].Workspace: {"ws-def456"},
		}

		triggers, err := MergeRunTriggers(inputs, wsInputs, workspaces)
		assert.NoError(t, err)

		assert.Equal(t, RunTriggers{
			{SourceID: "ws-def456", Workspace: workspaces[0]},
		}, triggers)
	})
}
