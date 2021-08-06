package main

type TeamAccess struct {
	Access        string                 `yaml:"access,omitempty"`
	Permissions   *TeamAccessPermissions `yaml:"permissions,omitempty"`
	TeamName      string                 `yaml:"team_name"`
	ResourceName  string                 `yaml:"resource_name"`
	WorkspaceName string
}

type TeamAccessPermissions struct {
	Runs             string `yaml:"runs" json:"runs"`
	Variables        string `yaml:"variables" json:"variables"`
	StateVersions    string `yaml:"state_versions" json:"state_versions"`
	SentinelMocks    string `yaml:"sentinel_mocks" json:"sentinel_mocks"`
	WorkspaceLocking bool   `yaml:"workspace_locking" json:"workspace_locking"`
}

// MergeWorkspaceIDs returns a new slice of TeamAccess structs
func MergeWorkspaceIDs(teamAccess []TeamAccess, workspaceNames []string) []TeamAccess {
	ts := make([]TeamAccess, len(teamAccess)*len(workspaceNames))

	i := 0
	for _, team := range teamAccess {
		for _, name := range workspaceNames {
			team.WorkspaceName = name
			ts[i] = team
			i = i + 1
		}
	}

	return ts
}

func (ta TeamAccess) GetResourceName() string {
	if ta.ResourceName != "" {
		return ta.ResourceName
	}

	return ta.TeamName
}
