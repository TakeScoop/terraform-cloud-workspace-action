package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeWorkspaceIDs(t *testing.T) {
	assert.Equal(t,
		MergeWorkspaceIDs(
			map[string]TeamAccess{
				"readers": {Access: "read", TeamName: "readers"},
				"writers": {Access: "write", TeamName: "writers"},
			},
			[]*Workspace{
				{Name: "api-staging", Workspace: "staging"},
				{Name: "api-production", Workspace: "staging"},
			},
		),
		[]TeamAccess{
			{Access: "read", TeamName: "readers", WorkspaceName: "api-staging", ResourceName: "readers"},
			{Access: "read", TeamName: "readers", WorkspaceName: "api-production", ResourceName: "readers"},
			{Access: "write", TeamName: "writers", WorkspaceName: "api-staging", ResourceName: "writers"},
			{Access: "write", TeamName: "writers", WorkspaceName: "api-production", ResourceName: "writers"},
		},
	)
}
