package action

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

type VariablesInput []VariablesInputItem

type WorkspaceVariablesInput map[string]VariablesInput

type VariablesInputItem struct {
	Key         string `yaml:"key"`
	Value       string `yaml:"value"`
	Description string `yaml:"description,omitempty"`
	Category    string `yaml:"category,omitempty"`
	Sensitive   bool   `yaml:"sensitive,omitempty"`
}

type Variables []Variable

type Variable struct {
	Key         string
	Value       string
	Description string
	Category    string
	Sensitive   bool
	Workspace   *Workspace
}

// NewVariable creates a new Variable struct
func NewVariable(vi VariablesInputItem, w *Workspace) *Variable {
	return &Variable{
		Key:         vi.Key,
		Value:       vi.Value,
		Description: vi.Description,
		Category:    vi.Category,
		Sensitive:   vi.Sensitive,
		Workspace:   w,
	}
}

// ToResource converts a variable to a Terraform variable resource
func (v Variable) ToResource() *tfeprovider.Variable {
	return &tfeprovider.Variable{
		Key:         v.Key,
		Value:       v.Value,
		Description: v.Description,
		Category:    v.Category,
		Sensitive:   v.Sensitive,
		WorkspaceID: fmt.Sprintf("${tfe_workspace.workspace[%q].id}", v.Workspace.Workspace),
	}
}

// ToVariable takes a tfe.Variable and returns a Variable
func ToVariable(v *tfe.Variable, workspace *Workspace) *Variable {
	return &Variable{
		Key:         v.Key,
		Value:       v.Value,
		Description: v.Description,
		Category:    string(v.Category),
		Sensitive:   v.Sensitive,
		Workspace:   workspace,
	}
}

// FetchRelatedVariables returns tfe.Variables related to the passed workspace
func FetchRelatedVariables(ctx context.Context, client *tfe.Client, workspace *Workspace) ([]*tfe.Variable, error) {
	vars, err := client.Variables.List(ctx, *workspace.ID, tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: maxPageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	return vars.Items, nil
}
