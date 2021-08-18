package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeWorkspaceIDs(t *testing.T) {
	assert.Equal(t,
		MergeWorkspaceIDs(
			TeamAccessInput{
				{Access: "read", TeamName: "readers"},
				{Access: "write", TeamName: "writers"},
			},
			[]*Workspace{
				{Name: "api-staging", Workspace: "staging"},
				{Name: "api-production", Workspace: "staging"},
			},
		),
		TeamAccessInput{
			{Access: "read", TeamName: "readers", WorkspaceName: "api-staging"},
			{Access: "read", TeamName: "readers", WorkspaceName: "api-production"},
			{Access: "write", TeamName: "writers", WorkspaceName: "api-staging"},
			{Access: "write", TeamName: "writers", WorkspaceName: "api-production"},
		},
	)
}

func TestTeamAccessValidate(t *testing.T) {
	t.Run("valid with team name", func(t *testing.T) {
		access := TeamAccessInputItem{TeamName: "foo", WorkspaceName: "workspace"}
		assert.NoError(t, access.Validate())
	})

	t.Run("valid with team ID", func(t *testing.T) {
		access := TeamAccessInputItem{TeamID: "team-abc123", WorkspaceName: "workspace"}
		assert.NoError(t, access.Validate())
	})

	t.Run("not valid with team ID and team name", func(t *testing.T) {
		access := TeamAccessInputItem{TeamName: "foo", TeamID: "team-abc123", WorkspaceName: "workspace"}
		assert.Error(t, access.Validate())
	})
}
