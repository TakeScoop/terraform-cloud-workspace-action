package action

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestFindRelatedTeamAccess(t *testing.T) {

	t.Run("team access found", func(t *testing.T) {
		ctx := context.Background()

		mux := http.NewServeMux()
		server := httptest.NewServer(mux)

		t.Cleanup(func() {
			server.Close()
		})

		mux.HandleFunc("/api/v2/organizations/org/teams", testServerResHandler(t, 200, teamAPIResponse))
		mux.HandleFunc("/api/v2/team-workspaces", testServerResHandler(t, 200, teamAccessAPIResponse))

		client := newTestTFClient(t, server.URL)

		workspace := &Workspace{Name: "ws", Workspace: "default", ID: strPtr("ws-abc123")}
		access, err := FindRelatedTeamAccess(ctx, client, workspace, "org")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(access), 1)
		assert.Equal(t, access, TeamAccess{
			{
				TeamName:  "Readers",
				Access:    "write",
				Workspace: workspace,
			},
		})
	})

	t.Run("no team access found", func(t *testing.T) {
		ctx := context.Background()

		mux := http.NewServeMux()
		server := httptest.NewServer(mux)

		t.Cleanup(func() {
			server.Close()
		})

		mux.HandleFunc("/api/v2/organizations/org/teams", testServerResHandler(t, 200, `{"data": []}`))
		mux.HandleFunc("/api/v2/team-workspaces", testServerResHandler(t, 200, `{"data": []}`))

		client := newTestTFClient(t, server.URL)

		workspace := &Workspace{Name: "ws", Workspace: "default", ID: strPtr("ws-abc123")}
		access, err := FindRelatedTeamAccess(ctx, client, workspace, "org")
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(access), 0)
	})
}
