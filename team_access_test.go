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
