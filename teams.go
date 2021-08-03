package main

import "fmt"

type Team struct {
	TeamID      string           `json:"team_id"`
	WorkspaceID string           `json:"workspace_id"`
	Access      string           `json:"access,omitempty"`
	Permissions *TeamPermissions `json:"permissions,omitempty"`
}

type TeamPermissions struct {
	Runs             string `json:"runs"`
	Variables        string `json:"variables"`
	StateVersions    string `json:"state_versions"`
	SentinelMocks    string `json:"sentinel_mocks"`
	WorkspaceLocking bool   `json:"workspace_locking"`
}

func (t *Team) Validate() error {
	if t.Access == "" && t.Permissions == nil {
		return fmt.Errorf("team access or permissions must be set")
	}

	if t.Permissions != nil && t.Access != "" {
		return fmt.Errorf("team access or permissions cannot both be set")
	}

	if t.TeamID == "" {
		return fmt.Errorf("team ID required")
	}

	if t.WorkspaceID == "" {
		return fmt.Errorf("team workspace ID required")
	}

	return nil
}
