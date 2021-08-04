package main

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestMergeWorkspaceIDs(t *testing.T) {
	assert.DeepEqual(t,
		MergeWorkspaceIDs(
			[]TeamAccess{
				{Access: "read", TeamName: "readers"},
				{Access: "write", TeamName: "writers"},
			},
			[]string{"staging", "production"},
		),
		[]TeamAccess{
			{Access: "read", TeamName: "readers", WorkspaceName: "staging"},
			{Access: "read", TeamName: "readers", WorkspaceName: "production"},
			{Access: "write", TeamName: "writers", WorkspaceName: "staging"},
			{Access: "write", TeamName: "writers", WorkspaceName: "production"},
		},
	)
}
