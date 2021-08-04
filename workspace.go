package main

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
)

type WorkspaceConfig struct {
	Terraform WorkspaceTerraform                `json:"terraform"`
	Variables map[string]WorkspaceVariable      `json:"variable,omitempty"`
	Resources map[string]map[string]interface{} `json:"resource,omitempty"`
	Data      map[string]map[string]interface{} `json:"data,omitempty"`
}

type WorkspaceBackend struct {
	S3 S3BackendConfig `json:"s3,omitempty"`
}

type WorkspaceTerraform struct {
	Backend WorkspaceBackend `json:"backend"`
}

type S3BackendConfig struct{}

type WorkspaceVariable struct {
	Type        string      `json:"type,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

type WorkspaceVCSBlock struct {
	OauthTokenID      string `json:"oauth_token_id"`
	Identifier        string `json:"identifier"`
	IngressSubmodules bool   `json:"ingress_submodules"`
}

type RemoteStateBackendConfigWorkspaces struct {
	Name string `json:"name"`
}

type RemoteStateBackendConfig struct {
	Key          *string                             `json:"key,omitempty"`
	Bucket       *string                             `json:"bucket,omitempty"`
	Region       *string                             `json:"region,omitempty"`
	Hostname     *string                             `json:"hostname,omitempty"`
	Organization *string                             `json:"organization,omitempty"`
	Workspaces   *RemoteStateBackendConfigWorkspaces `json:"workspaces,omitempty"`
}

type RemoteStateBlock struct {
	Config  RemoteStateBackendConfig `json:"config" yaml:"config"`
	Backend string                   `json:"backend" yaml:"backend"`
}

type WorkspaceWorkspaceResource struct {
	ForEach                string             `json:"for_each,omitempty"`
	AgentPoolID            string             `json:"agent_pool_id,omitempty"`
	AutoApply              *bool              `json:"auto_apply,omitempty"`
	ExecutionMode          string             `json:"execution_mode,omitempty"`
	FileTriggersEnabled    *bool              `json:"file_triggers_enabled,omitempty"`
	GlobalRemoteState      *bool              `json:"global_remote_state,omitempty"`
	Name                   string             `json:"name"`
	Organization           string             `json:"organization"`
	QueueAllRuns           *bool              `json:"queue_all_runs,omitempty"`
	RemoteStateConsumerIDs []string           `json:"remote_state_consumer_ids,omitempty"`
	SpeculativeEnabled     *bool              `json:"speculative_enabled,omitempty"`
	TerraformVersion       string             `json:"terraform_version,omitempty"`
	SSHKeyID               string             `json:"ssh_key_id,omitempty"`
	VCSRepo                *WorkspaceVCSBlock `json:"vcs_repo,omitempty"`
	WorkingDirectory       string             `json:"working_directory,omitempty"`
}

type WorkspaceVariableResource struct {
	ForEach     string `json:"for_each,omitempty"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}

// getVCSClientByName looks for a VCS client of the passed type against the VCS clients in the Terraform Cloud organization
func getVCSClientByName(ctx context.Context, tfc *tfe.Client, organization string, vcsType string) (*tfe.OAuthClient, error) {
	list, err := tfc.OAuthClients.List(ctx, organization, tfe.OAuthClientListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, v := range list.Items {
		if v.ServiceProvider == tfe.ServiceProviderType(vcsType) {
			return v, nil
		}
	}

	return nil, fmt.Errorf("no VCS client found of type %s", vcsType)
}

// GetVCSTokenIDByClientType returns an OAuth client token ID for the passed VCS type
func GetVCSTokenIDByClientType(ctx context.Context, tfc *tfe.Client, organization string, vcsType string) (string, error) {
	vcsClient, err := getVCSClientByName(ctx, tfc, organization, vcsType)
	if err != nil {
		return "", err
	}

	if len(vcsClient.OAuthTokens) == 0 {
		return "", fmt.Errorf("no VCS tokens found for client %s:%s", vcsClient.ServiceProviderName, vcsClient.ID)
	}

	return vcsClient.OAuthTokens[0].ID, nil
}

type WorkspaceConfigOptions struct {
	AgentPoolID            string
	AutoApply              *bool
	ExecutionMode          string
	FileTriggersEnabled    *bool
	GlobalRemoteState      *bool
	Organization           string
	QueueAllRuns           *bool
	RemoteStateConsumerIDs string
	SpeculativeEnabled     *bool
	SSHKeyID               string
	TerraformVersion       string
	VCSIngressSubmodules   bool
	VCSRepo                string
	VCSTokenID             string
	VCSType                string
	WorkingDirectory       string
}

// NewWorkspaceResource adds defaults and conditional fields to a WorkspaceWorkspaceResource struct
func NewWorkspaceResource(ctx context.Context, client *tfe.Client, config WorkspaceConfigOptions) (*WorkspaceWorkspaceResource, error) {
	ws := &WorkspaceWorkspaceResource{
		ForEach:      "${var.workspace_names}",
		Name:         "${each.value}",
		Organization: config.Organization,
	}

	if config.AutoApply != nil {
		ws.AutoApply = config.AutoApply
	}

	var vcs *WorkspaceVCSBlock

	if config.VCSType != "" || config.VCSTokenID != "" {
		if config.VCSRepo == "" {
			return nil, fmt.Errorf("VCS repository must be passed if VCS type or a VCS token ID is passed")
		}

		vcsTokenID := config.VCSTokenID
		if vcsTokenID == "" {
			t, err := GetVCSTokenIDByClientType(ctx, client, config.Organization, config.VCSType)
			if err != nil {
				return nil, err
			}

			vcsTokenID = t
		} else {
			vcsTokenID = config.VCSTokenID
		}

		vcs = &WorkspaceVCSBlock{
			OauthTokenID:      vcsTokenID,
			Identifier:        config.VCSRepo,
			IngressSubmodules: config.VCSIngressSubmodules,
		}
	}

	ws.VCSRepo = vcs

	if config.AgentPoolID != "" {
		ws.AgentPoolID = config.AgentPoolID
		ws.ExecutionMode = "agent"
	} else if config.ExecutionMode != "" {
		ws.ExecutionMode = config.ExecutionMode
	}

	if config.GlobalRemoteState != nil {
		if !*config.GlobalRemoteState {
			ws.GlobalRemoteState = config.GlobalRemoteState
			ws.RemoteStateConsumerIDs = strings.FieldsFunc(config.RemoteStateConsumerIDs, func(c rune) bool { return c == ',' })
		}
	}

	ws.TerraformVersion = config.TerraformVersion
	ws.QueueAllRuns = config.QueueAllRuns
	ws.SpeculativeEnabled = config.SpeculativeEnabled
	ws.FileTriggersEnabled = config.FileTriggersEnabled
	ws.SSHKeyID = config.SSHKeyID
	ws.WorkingDirectory = config.WorkingDirectory

	return ws, nil
}

// NewWorkspaceVariableResource takes a Variable and uses it to fill a new WorkspaceVariableResource
func NewWorkspaceVariableResource(v *Variable) *WorkspaceVariableResource {
	return &WorkspaceVariableResource{
		Key:         v.Key,
		Value:       v.Value,
		Description: v.Description,
		Category:    v.Category,
		Sensitive:   v.Sensitive,
		WorkspaceID: fmt.Sprintf("${tfe_workspace.workspace[%q].id}", v.WorkspaceName),
	}
}
