package main

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type WorkspaceTestCase struct {
	Name               string
	Workspaces         []*Workspace
	Variables          []Variable
	WorkspaceVariables map[string][]Variable
	AssertEqual        []Variable
}

func sortVariables(slice []Variable) {
	sort.Slice(slice[:], func(i, j int) bool {
		return fmt.Sprintf("%s-%s", slice[i].WorkspaceName, slice[i].Key) < fmt.Sprintf("%s-%s", slice[j].WorkspaceName, slice[j].Key)
	})
}

func TestParseVariablesByWorkspace(t *testing.T) {
	workspaceTestCases := []WorkspaceTestCase{
		{
			Name:               "apply single variable to single workspace",
			Workspaces:         []*Workspace{{Name: "app", Workspace: "default"}},
			Variables:          []Variable{{Key: "foo", Value: "bar"}},
			WorkspaceVariables: map[string][]Variable{},
			AssertEqual: []Variable{{
				Key:           "foo",
				Value:         "bar",
				WorkspaceName: "app",
				Category:      "env",
			}},
		},
		{
			Name:       "apply multiple variables to single workspace",
			Workspaces: []*Workspace{{Name: "app", Workspace: "default"}},
			Variables: []Variable{
				{Key: "foo", Value: "bar"},
				{Key: "baz", Value: "woz"},
			},
			WorkspaceVariables: map[string][]Variable{},
			AssertEqual: []Variable{
				{
					Key:           "baz",
					Value:         "woz",
					WorkspaceName: "app",
					Category:      "env",
				},
				{
					Key:           "foo",
					Value:         "bar",
					WorkspaceName: "app",
					Category:      "env",
				},
			},
		},
		{
			Name:               "apply nothing when neither variables nor workspace variables are passed",
			Workspaces:         []*Workspace{{Name: "app", Workspace: "default"}},
			Variables:          []Variable{},
			WorkspaceVariables: map[string][]Variable{},
			AssertEqual:        []Variable{},
		},
		{
			Name: "apply variables to all workspaces",
			Workspaces: []*Workspace{
				{Name: "app-staging", Workspace: "staging"},
				{Name: "app-production", Workspace: "production"},
			},
			Variables:          []Variable{{Key: "foo", Value: "bar"}},
			WorkspaceVariables: map[string][]Variable{},
			AssertEqual: []Variable{
				{
					Key:           "foo",
					Value:         "bar",
					WorkspaceName: "app-staging",
					Category:      "env",
				},
				{
					Key:           "foo",
					Value:         "bar",
					WorkspaceName: "app-production",
					Category:      "env",
				},
			},
		},
		{
			Name: "apply workspace variables to named workspaces",
			Workspaces: []*Workspace{
				{Name: "app-staging", Workspace: "staging"},
				{Name: "app-production", Workspace: "production"},
			},
			Variables: []Variable{},
			WorkspaceVariables: map[string][]Variable{
				"staging": {
					{
						Key:   "environment",
						Value: "staging",
					},
				},
				"production": {
					{
						Key:   "environment",
						Value: "production",
					},
				},
			},
			AssertEqual: []Variable{
				{
					Key:           "environment",
					Value:         "staging",
					WorkspaceName: "app-staging",
					Category:      "env",
				},
				{
					Key:           "environment",
					Value:         "production",
					WorkspaceName: "app-production",
					Category:      "env",
				},
			},
		},
		{
			Name: "apply workspace variables to single workspaces",
			Workspaces: []*Workspace{
				{Name: "app-staging", Workspace: "staging"},
				{Name: "app-production", Workspace: "production"},
			},
			Variables: []Variable{},
			WorkspaceVariables: map[string][]Variable{
				"staging": {
					{
						Key:   "environment",
						Value: "staging",
					},
				},
			},
			AssertEqual: []Variable{
				{
					Key:           "environment",
					Value:         "staging",
					WorkspaceName: "app-staging",
					Category:      "env",
				},
			},
		},
	}

	for _, c := range workspaceTestCases {
		t.Run(c.Name, func(t *testing.T) {
			vars, err := ParseVariablesByWorkspace(
				c.Workspaces,
				&c.Variables,
				&c.WorkspaceVariables,
			)
			if err != nil {
				t.Fatal(err)
			}

			sortVariables(vars)
			sortVariables(c.AssertEqual)

			assert.Equal(t, vars, c.AssertEqual)
		})
	}

	t.Run("error when workspace variable workspace is not found in passed workspace names", func(t *testing.T) {
		_, err := ParseVariablesByWorkspace(
			[]*Workspace{
				{Name: "api", Workspace: "foo"},
			},
			&[]Variable{},
			&map[string][]Variable{
				"bar": {{
					Key:   "should",
					Value: "error",
				}},
			},
		)
		assert.EqualError(t, err, fmt.Sprintf("workspace %q was not found in planned workspaces [foo]", "bar"))
	})
}
