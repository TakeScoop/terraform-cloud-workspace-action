package tfeprovider

type Variable struct {
	ForEach     string `json:"for_each,omitempty"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}
