package tfconfig

type RemoteStateBackendConfigWorkspaces struct {
	Name string `json:"name"`
}

type RemoteStateBackendConfig struct {
	Key          string                              `json:"key,omitempty"`
	Bucket       string                              `json:"bucket,omitempty"`
	Region       string                              `json:"region,omitempty"`
	Hostname     string                              `json:"hostname,omitempty"`
	Organization string                              `json:"organization,omitempty"`
	Workspaces   *RemoteStateBackendConfigWorkspaces `json:"workspaces,omitempty"`
}

type RemoteState struct {
	Config  RemoteStateBackendConfig `json:"config" yaml:"config"`
	Backend string                   `json:"backend" yaml:"backend"`
}
