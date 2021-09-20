package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type NewTeamAccessTestCase struct {
	Description string
	Workspaces  []*Workspace
	Input       TeamAccessInput
	Expect      TeamAccess
}

func TestNewTeamAccess(t *testing.T) {
	for _, testCase := range []NewTeamAccessTestCase{
		{
			Description: "single access, single workspace",
			Workspaces: []*Workspace{
				{Name: "staging"},
			},
			Input: TeamAccessInput{
				TeamAccessInputItem{Access: "read", TeamName: "Readers"},
			},
			Expect: TeamAccess{
				TeamAccessItem{Access: "read", TeamName: "Readers", Workspace: &Workspace{Name: "staging"}},
			},
		},
		{
			Description: "single access, multi workspace",
			Workspaces: []*Workspace{
				{Name: "staging"},
				{Name: "production"},
			},
			Input: TeamAccessInput{
				TeamAccessInputItem{Access: "read", TeamName: "Readers"},
			},
			Expect: TeamAccess{
				TeamAccessItem{Access: "read", TeamName: "Readers", Workspace: &Workspace{Name: "staging"}},
				TeamAccessItem{Access: "read", TeamName: "Readers", Workspace: &Workspace{Name: "production"}},
			},
		},
		{
			Description: "multi access, single workspace",
			Workspaces: []*Workspace{
				{Name: "staging"},
			},
			Input: TeamAccessInput{
				TeamAccessInputItem{Access: "read", TeamName: "Readers"},
				TeamAccessInputItem{Access: "write", TeamName: "Writers"},
			},
			Expect: TeamAccess{
				TeamAccessItem{Access: "read", TeamName: "Readers", Workspace: &Workspace{Name: "staging"}},
				TeamAccessItem{Access: "write", TeamName: "Writers", Workspace: &Workspace{Name: "staging"}},
			},
		},
		{
			Description: "multi access, multi workspace",
			Workspaces: []*Workspace{
				{Name: "staging"},
				{Name: "production"},
			},
			Input: TeamAccessInput{
				TeamAccessInputItem{Access: "read", TeamName: "Readers"},
				TeamAccessInputItem{Access: "write", TeamName: "Writers"},
			},
			Expect: TeamAccess{
				TeamAccessItem{Access: "read", TeamName: "Readers", Workspace: &Workspace{Name: "staging"}},
				TeamAccessItem{Access: "read", TeamName: "Readers", Workspace: &Workspace{Name: "production"}},
				TeamAccessItem{Access: "write", TeamName: "Writers", Workspace: &Workspace{Name: "staging"}},
				TeamAccessItem{Access: "write", TeamName: "Writers", Workspace: &Workspace{Name: "production"}},
			},
		},
	} {
		t.Run(testCase.Description, func(t *testing.T) {
			access := NewTeamAccess(testCase.Input, testCase.Workspaces)
			assert.Equal(t, access, testCase.Expect)
		})
	}
}
