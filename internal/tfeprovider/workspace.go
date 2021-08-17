package tfeprovider

type Workspace struct {
	// TODO: handle for_each via an embedded struct instead of duplicating
	ForEach string `json:"for_each,omitempty"`

	AgentPoolID            string   `json:"agent_pool_id,omitempty"`
	AutoApply              *bool    `json:"auto_apply,omitempty"`
	ExecutionMode          string   `json:"execution_mode,omitempty"`
	FileTriggersEnabled    *bool    `json:"file_triggers_enabled,omitempty"`
	GlobalRemoteState      *bool    `json:"global_remote_state,omitempty"`
	Name                   string   `json:"name"`
	Organization           string   `json:"organization"`
	QueueAllRuns           *bool    `json:"queue_all_runs,omitempty"`
	RemoteStateConsumerIDs []string `json:"remote_state_consumer_ids,omitempty"`
	SpeculativeEnabled     *bool    `json:"speculative_enabled,omitempty"`
	TerraformVersion       string   `json:"terraform_version,omitempty"`
	SSHKeyID               string   `json:"ssh_key_id,omitempty"`
	VCSRepo                *VCSRepo `json:"vcs_repo,omitempty"`
	WorkingDirectory       string   `json:"working_directory,omitempty"`
}

type VCSRepo struct {
	OauthTokenID      string `json:"oauth_token_id"`
	Identifier        string `json:"identifier"`
	IngressSubmodules bool   `json:"ingress_submodules"`
}
