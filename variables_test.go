package main

import (
	"fmt"
	"sort"
	"testing"

	"gotest.tools/v3/assert"
)

type WorkspaceTestCase struct {
	Name               string
	Workspaces         []string
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
			Workspaces:         []string{"staging"},
			Variables:          []Variable{{Key: "foo", Value: "bar"}},
			WorkspaceVariables: map[string][]Variable{},
			AssertEqual: []Variable{{
				Key:           "foo",
				Value:         "bar",
				WorkspaceName: "staging",
				Category:      "env",
			}},
		},
		{
			Name:       "apply multiple variables to single workspace",
			Workspaces: []string{"staging"},
			Variables: []Variable{
				{Key: "foo", Value: "bar"},
				{Key: "baz", Value: "woz"},
			},
			WorkspaceVariables: map[string][]Variable{},
			AssertEqual: []Variable{
				{
					Key:           "baz",
					Value:         "woz",
					WorkspaceName: "staging",
					Category:      "env",
				},
				{
					Key:           "foo",
					Value:         "bar",
					WorkspaceName: "staging",
					Category:      "env",
				},
			},
		},
		{
			Name:               "apply nothing when variables or workspace variables are passed",
			Workspaces:         []string{"staging"},
			Variables:          []Variable{},
			WorkspaceVariables: map[string][]Variable{},
			AssertEqual:        []Variable{},
		},
		{
			Name:               "apply variables to all workspaces",
			Workspaces:         []string{"staging", "production"},
			Variables:          []Variable{{Key: "foo", Value: "bar"}},
			WorkspaceVariables: map[string][]Variable{},
			AssertEqual: []Variable{
				{
					Key:           "foo",
					Value:         "bar",
					WorkspaceName: "production",
					Category:      "env",
				},
				{
					Key:           "foo",
					Value:         "bar",
					WorkspaceName: "staging",
					Category:      "env",
				},
			},
		},
		{
			Name:       "apply workspace variables to named workspaces",
			Workspaces: []string{"staging", "production"},
			Variables:  []Variable{},
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
					WorkspaceName: "staging",
					Category:      "env",
				},
				{
					Key:           "environment",
					Value:         "production",
					WorkspaceName: "production",
					Category:      "env",
				},
			},
		},
		{
			Name:       "apply workspace variables to single workspaces",
			Workspaces: []string{"staging", "production"},
			Variables:  []Variable{},
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
					WorkspaceName: "staging",
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

			assert.DeepEqual(t, vars, c.AssertEqual)
		})
	}

	t.Run("error when workspace variable workspace is not found in passed workspace names", func(t *testing.T) {
		_, err := ParseVariablesByWorkspace(
			[]string{"foo"},
			&[]Variable{},
			&map[string][]Variable{
				"bar": {{
					Key:   "should",
					Value: "error",
				}},
			},
		)
		assert.ErrorContains(t, err, fmt.Sprintf("workspace %q was not found", "bar"))
	})
}
