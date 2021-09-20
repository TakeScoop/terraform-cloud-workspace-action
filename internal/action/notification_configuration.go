package action

import "github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"

type NotificationInput struct {
	Name            string `yaml:"name"`
	DestinationType string `yaml:"destination_type"`

	URL            string   `yaml:"url,omitempty"`
	EmailAddresses []string `yaml:"email_addresses,omitempty"`
	EmailUserIDs   []string `yaml:"email_user_ids,omitempty"`
	Enabled        string   `yaml:"enabled,omitempty"`
	Token          string   `yaml:"token,omitempty"`
	Triggers       []string `yaml:"triggers,omitempty"`
}

type Notification struct {
	Input     NotificationInput
	Workspace *Workspace
}

// MergeNotifications returns a list of notifications, one notification object per workspace
func MergeNotifications(input NotificationInput, workspaces []*Workspace) (notifications []*Notification) {
	for _, ws := range workspaces {
		notifications = append(notifications, &Notification{
			Input:     input,
			Workspace: ws,
		})
	}

	return notifications
}

func (n Notification) ToResource() *tfeprovider.NotificationConfiguration {
	return &tfeprovider.NotificationConfiguration{
		Name:            n.Input.Name,
		DestinationType: n.Input.DestinationType,
		URL:             n.Input.URL,
		WorkspaceID:     *n.Workspace.ID,
		EmailAddresses:  n.Input.EmailAddresses,
		EmailUserIDs:    n.Input.EmailUserIDs,
		Enabled:         n.Input.Enabled,
		Token:           n.Input.Token,
		Triggers:        n.Input.Triggers,
	}
}
