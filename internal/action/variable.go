package action

import (
	"fmt"

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

type Variables []VariablesItem

type VariablesItem struct {
	Key         string
	Value       string
	Description string
	Category    string
	Sensitive   bool
	Workspace   *Workspace
}

func NewVariablesItem(vi VariablesInputItem, w *Workspace) *VariablesItem {
	return &VariablesItem{
		Key:         vi.Key,
		Value:       vi.Value,
		Description: vi.Description,
		Category:    vi.Category,
		Sensitive:   vi.Sensitive,
		Workspace:   w,
	}
}

func (avi VariablesItem) ToResource() *tfeprovider.Variable {
	resource := &tfeprovider.Variable{
		Key:         avi.Key,
		Value:       avi.Value,
		Description: avi.Description,
		Category:    avi.Category,
		Sensitive:   avi.Sensitive,
		WorkspaceID: fmt.Sprintf("${tfe_workspace.workspace[%q].id}", avi.Workspace.Name),
	}

	if resource.Category == "" {
		resource.Category = "env"
	}

	return resource
}

func (av Variables) ToResource() []*tfeprovider.Variable {
	vars := make([]*tfeprovider.Variable, len(av))

	for i, avi := range av {
		vars[i] = avi.ToResource()
	}

	return vars
}
