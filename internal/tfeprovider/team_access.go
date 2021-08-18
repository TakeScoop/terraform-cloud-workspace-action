package tfeprovider

type TeamAccess struct {
	ForEach     map[string]TeamAccess  `json:"for_each,omitempty"`
	TeamID      string                 `json:"team_id"`
	WorkspaceID string                 `json:"workspace_id"`
	Access      string                 `json:"access,omitempty"`
	Permissions *TeamAccessPermissions `json:"permissions,omitempty"`

	// TODO: Allow resources to support dynamic attrs via embedded structs or methods
	// Avoid these awkwardly named structs/exposed implementation details
	DynamicPermissions *DynamicPermissions `json:"dynamic,omitempty"`
}

type TeamAccessPermissions struct {
	Runs             string      `json:"runs"`
	Variables        string      `json:"variables"`
	StateVersions    string      `json:"state_versions"`
	SentinelMocks    string      `json:"sentinel_mocks"`
	WorkspaceLocking interface{} `json:"workspace_locking"`
}

type DynamicPermissions struct {
	Permission []DynamicPermissionEntry `json:"permissions,omitempty"`
}

type DynamicPermissionEntry struct {
	ForEach string                 `json:"for_each"`
	Content *TeamAccessPermissions `json:"content"`
}
