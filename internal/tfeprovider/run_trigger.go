package tfeprovider

type RunTrigger struct {
	ForEach      map[string]RunTrigger `json:"for_each,omitempty"`
	WorkspaceID  string                `json:"workspace_id"`
	SourceableID string                `json:"sourceable_id"`
}
