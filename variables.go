package main

import (
	"fmt"
)

type Variable struct {
	Key           string `yaml:"key"`
	Value         string `yaml:"value"`
	Description   string `yaml:"description,omitempty"`
	Category      string `yaml:"category,omitempty"`
	Sensitive     bool   `yaml:"sensitive,omitempty"`
	WorkspaceName string
}

func contains(strings []string, target string) bool {
	for _, v := range strings {
		if v == target {
			return true
		}
	}
	return false
}

// ParseVariablesByWorkspace takes a list of workspace names, general variables and workspaced variables and flattens them into a single set
func ParseVariablesByWorkspace(names []string, generalVars *[]Variable, workspaceVars *map[string][]Variable) ([]Variable, error) {
	vars := []Variable{}
	for _, v := range *generalVars {
		for _, ws := range names {
			newVar := v

			newVar.WorkspaceName = ws

			vars = append(vars, newVar)
		}
	}

	for ws, vs := range *workspaceVars {
		if !contains(names, ws) {
			return nil, fmt.Errorf("workspace %q was not found in planned workspaces %v", ws, names)
		}

		for _, v := range vs {
			v.WorkspaceName = ws

			vars = append(vars, v)
		}
	}

	for i := range vars {
		if vars[i].Category == "" {
			vars[i].Category = "env"
		}
	}

	return vars, nil
}
