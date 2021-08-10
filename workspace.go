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
	Providers map[string]WorkspaceProvider      `json:"provider"`
}

type WorkspaceTerraform struct {
	Backend WorkspaceBackend `json:"backend"`
}

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

type WorkspaceProvider struct {
	Version  string `json:"version"`
	Hostname string `json:"hostname"`
	Token    string `json:"token,omitempty"`
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

type WorkspaceResourceOptions struct {
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
func NewWorkspaceResource(ctx context.Context, client *tfe.Client, config *WorkspaceResourceOptions) (*WorkspaceWorkspaceResource, error) {
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

// AddVariable adds the passed variables to the calling workspace
func (ws *WorkspaceConfig) AddVariables(vars []Variable) {
	if len(vars) == 0 {
		return
	}

	varResources := map[string]interface{}{}

	for _, v := range vars {
		varResources[fmt.Sprintf("%s-%s", v.WorkspaceName, v.Key)] = NewWorkspaceVariableResource(v)
	}

	ws.Resources["tfe_variable"] = varResources
}

// NewWorkspaceVariableResource takes a Variable and uses it to fill a new WorkspaceVariableResource
func NewWorkspaceVariableResource(v Variable) *WorkspaceVariableResource {
	return &WorkspaceVariableResource{
		Key:         v.Key,
		Value:       v.Value,
		Description: v.Description,
		Category:    v.Category,
		Sensitive:   v.Sensitive,
		WorkspaceID: fmt.Sprintf("${tfe_workspace.workspace[%q].id}", v.WorkspaceName),
	}
}

type WorkspaceTeamAccessResource struct {
	TeamID      string                 `json:"team_id"`
	WorkspaceID string                 `json:"workspace_id"`
	Access      string                 `json:"access,omitempty"`
	Permissions *TeamAccessPermissions `json:"permissions,omitempty"`
}

// NewWorkspaceTeamAccessResource takes a Team object and uses it to fill a new WorkspaceTeamAccessResource
func NewWorkspaceTeamAccessResource(ta *TeamAccess) *WorkspaceTeamAccessResource {
	return &WorkspaceTeamAccessResource{
		TeamID:      fmt.Sprintf("${data.tfe_team.%s.id}", ta.TeamName),
		WorkspaceID: fmt.Sprintf("${tfe_workspace.workspace[%q].id}", ta.WorkspaceName),
		Access:      ta.Access,
		Permissions: ta.Permissions,
	}
}

type TeamDataResource struct {
	Name         string `json:"name"`
	Organization string `json:"organization"`
}

// AddTeamAccess adds the passed teams to the calling workspace
func (ws *WorkspaceConfig) AddTeamAccess(teamAccess []TeamAccess, organization string) {
	if len(teamAccess) == 0 {
		return
	}

	ws.Data["tfe_team"] = map[string]interface{}{}
	ws.Resources["tfe_team_access"] = map[string]interface{}{}

	for _, ta := range teamAccess {

		_, ok := ws.Data["tfe_team"][ta.TeamName]
		if !ok {
			ws.Data["tfe_team"][ta.TeamName] = TeamDataResource{
				Name:         ta.TeamName,
				Organization: organization,
			}
		}

		ws.Resources["tfe_team_access"][fmt.Sprintf("%s-%s", ta.WorkspaceName, ta.TeamName)] = NewWorkspaceTeamAccessResource(&ta)
	}
}

// AddRemoteStates adds the passed remote state to the calling workspace
func (ws *WorkspaceConfig) AddRemoteStates(remoteStates map[string]RemoteState) {
	if len(remoteStates) == 0 {
		return
	}

	ws.Data["terraform_remote_state"] = map[string]interface{}{}

	for name, block := range remoteStates {
		ws.Data["terraform_remote_state"][name] = block
	}
}

type NewWorkspaceConfigOptions struct {
	TerraformBackendConfig   *WorkspaceTerraform
	WorkspaceVariables       map[string]WorkspaceVariable
	RemoteStates             map[string]RemoteState
	Variables                []Variable
	TeamAccess               []TeamAccess
	WorkspaceResourceOptions *WorkspaceResourceOptions
	Providers                map[string]WorkspaceProvider
}

// NewWorkspaceConfig takes in all required values for the Terraform workspace and outputs a struct that can be marshalled then planned or applied
func NewWorkspaceConfig(ctx context.Context, client *tfe.Client, config *NewWorkspaceConfigOptions) (*WorkspaceConfig, error) {
	wsResource, err := NewWorkspaceResource(ctx, client, config.WorkspaceResourceOptions)
	if err != nil {
		return nil, err
	}

	wsConfig := &WorkspaceConfig{
		Terraform: *config.TerraformBackendConfig,
		Variables: config.WorkspaceVariables,
		Data:      map[string]map[string]interface{}{},
		Resources: map[string]map[string]interface{}{
			"tfe_workspace": {
				"workspace": wsResource,
			},
		},
		Providers: config.Providers,
	}

	wsConfig.AddRemoteStates(config.RemoteStates)
	wsConfig.AddVariables(config.Variables)
	wsConfig.AddTeamAccess(config.TeamAccess, wsResource.Organization)

	return wsConfig, nil
}
