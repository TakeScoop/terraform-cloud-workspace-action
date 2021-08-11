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

func findWorkspace(workspaces []*Workspace, target string) *Workspace {
	for _, v := range workspaces {
		if v.Workspace == target {
			return v
		}
	}
	return nil
}

// ParseVariablesByWorkspace takes a list of workspace names, general variables and workspaced variables and flattens them into a single set
func ParseVariablesByWorkspace(workspaces []*Workspace, generalVars *[]Variable, workspaceVars *map[string][]Variable) ([]Variable, error) {
	vars := []Variable{}
	for _, v := range *generalVars {
		for _, ws := range workspaces {
			newVar := v

			newVar.WorkspaceName = ws.Name

			vars = append(vars, newVar)
		}
	}

	workspacesNames := make([]string, len(workspaces))
	for i, ws := range workspaces {
		workspacesNames[i] = ws.Workspace
	}

	for wsName, vs := range *workspaceVars {
		w := findWorkspace(workspaces, wsName)
		if w == nil {
			return nil, fmt.Errorf("workspace %q was not found in planned workspaces %v", wsName, workspacesNames)
		}

		for _, v := range vs {
			v.WorkspaceName = w.Name

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
