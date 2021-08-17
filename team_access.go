package main

import "fmt"

type TeamAccess struct {
	Access        string                 `yaml:"access,omitempty"`
	Permissions   *TeamAccessPermissions `yaml:"permissions,omitempty"`
	TeamName      string                 `yaml:"name"`
	TeamID        string                 `yaml:"id"`
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
func MergeWorkspaceIDs(teamAccess []TeamAccess, workspaces []*Workspace) []TeamAccess {
	ts := make([]TeamAccess, len(teamAccess)*len(workspaces))

	i := 0

	for _, team := range teamAccess {
		for _, ws := range workspaces {
			team.WorkspaceName = ws.Name
			ts[i] = team
			i = i + 1
		}
	}

	return ts
}

func (ta TeamAccess) Validate() error {
	if ta.TeamID != "" && ta.TeamName != "" {
		return fmt.Errorf("team name and team ID cannot both be set: %s, %s", ta.TeamID, ta.TeamName)
	}

	return nil
}
