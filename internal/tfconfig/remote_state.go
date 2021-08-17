package tfconfig

// AddRemoteStates adds remote state data sources to the module
// TODO: discard this entirely in favor of AppendResource
// also: the logic here is incorrect, this func is setting, it is not adding/appending
func (m *Module) AddRemoteStates(remoteStates map[string]RemoteState) {
	if len(remoteStates) == 0 {
		return
	}

	m.Data["terraform_remote_state"] = map[string]interface{}{}

	for name, block := range remoteStates {
		m.Data["terraform_remote_state"][name] = block
	}
}

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
