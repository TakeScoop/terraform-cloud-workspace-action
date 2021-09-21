package tfeprovider

type NotificationConfiguration struct {
	Name            string `json:"name"`
	DestinationType string `json:"destination_type"`
	WorkspaceID     string `json:"workspace_id"`

	URL            string   `json:"url,omitempty"`
	EmailAddresses []string `json:"email_addresses,omitempty"`
	EmailUserIDs   []string `json:"email_user_ids,omitempty"`
	Enabled        string   `json:"enabled,omitempty"`
	Token          string   `json:"token,omitempty"`
	Triggers       []string `json:"triggers,omitempty"`
}
