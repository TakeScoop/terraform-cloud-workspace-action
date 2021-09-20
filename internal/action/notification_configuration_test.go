package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

func TestNotificationToResource(t *testing.T) {
	t.Run("convert a notification", func(t *testing.T) {
		n := Notification{
			Input: NotificationInput{
				Name:            "foo",
				DestinationType: "email",
			},
			Workspace: newTestWorkspace(),
		}

		assert.Equal(t, &tfeprovider.NotificationConfiguration{
			Name:            "foo",
			DestinationType: "email",
			WorkspaceID:     "ws-abc123",
		}, n.ToResource())
	})
}

func TestMergeNotifications(t *testing.T) {
	t.Run("return a notification list", func(t *testing.T) {
		input := NotificationInput{
			Name:            "foo",
			DestinationType: "email",
		}

		workspaces := newTestMultiWorkspaceList()

		notifications := MergeNotifications(input, workspaces)

		assert.Len(t, notifications, 2)
		assert.Equal(t, []*Notification{
			{Input: input, Workspace: workspaces[0]},
			{Input: input, Workspace: workspaces[1]},
		}, notifications)
	})
}
