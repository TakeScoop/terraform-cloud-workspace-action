package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeWorkspaceIDs(t *testing.T) {
	assert.Equal(t,
		MergeWorkspaceIDs(
			[]TeamAccess{
				{Access: "read", TeamName: "readers"},
				{Access: "write", TeamName: "writers"},
			},
			[]*Workspace{
				{Name: "api-staging", Workspace: "staging"},
				{Name: "api-production", Workspace: "staging"},
			},
		),
		[]TeamAccess{
			{Access: "read", TeamName: "readers", WorkspaceName: "api-staging"},
			{Access: "read", TeamName: "readers", WorkspaceName: "api-production"},
			{Access: "write", TeamName: "writers", WorkspaceName: "api-staging"},
			{Access: "write", TeamName: "writers", WorkspaceName: "api-production"},
		},
	)
}
