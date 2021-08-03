package main

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestTeamValidate(t *testing.T) {
	t.Run("basic team with access set", func(t *testing.T) {
		team := &Team{
			TeamID:      "123",
			WorkspaceID: "456",
			Access:      "read",
		}

		assert.NilError(t, team.Validate())
	})

	t.Run("team with permissions set", func(t *testing.T) {
		team := &Team{
			TeamID:      "123",
			WorkspaceID: "456",
			Permissions: &TeamPermissions{
				Runs:             "none",
				Variables:        "none",
				StateVersions:    "none",
				SentinelMocks:    "none",
				WorkspaceLocking: false,
			},
		}

		assert.NilError(t, team.Validate())
	})

	t.Run("assert access or permissions must be set", func(t *testing.T) {
		team := &Team{
			TeamID:      "123",
			WorkspaceID: "456",
		}

		assert.Error(t, team.Validate(), "team access or permissions must be set")
	})

	t.Run("assert TeamID must be set", func(t *testing.T) {
		team := &Team{
			WorkspaceID: "456",
			Access:      "read",
		}

		assert.Error(t, team.Validate(), "team ID required")
	})

	t.Run("assert WorkspaceID must be set", func(t *testing.T) {
		team := &Team{
			TeamID: "123",
			Access: "read",
		}

		assert.Error(t, team.Validate(), "team workspace ID required")
	})

}
