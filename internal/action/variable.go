package action

type VariablesInput []VariablesInputItem

type WorkspaceVariablesInput map[string]VariablesInput

type VariablesInputItem struct {
	Key         string `yaml:"key"`
	Value       string `yaml:"value"`
	Description string `yaml:"description,omitempty"`
	Category    string `yaml:"category,omitempty"`
	Sensitive   bool   `yaml:"sensitive,omitempty"`

	// TODO: remove this, objects should not store data on behalf of callers
	WorkspaceName string
}
