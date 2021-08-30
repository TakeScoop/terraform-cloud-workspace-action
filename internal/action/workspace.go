package action

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfconfig"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

type Workspace struct {
	Name      string
	Workspace string
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
func NewWorkspaceResource(ctx context.Context, client *tfe.Client, workspaces []*Workspace, config *WorkspaceResourceOptions) (*tfeprovider.Workspace, error) {
	wsForEach := map[string]*tfeprovider.Workspace{}

	for _, ws := range workspaces {
		wsForEach[ws.Name] = &tfeprovider.Workspace{
			Name: ws.Name,
		}
	}

	ws := &tfeprovider.Workspace{
		ForEach:      wsForEach,
		Name:         "${each.value.name}",
		Organization: config.Organization,
	}

	if config.AutoApply != nil {
		ws.AutoApply = config.AutoApply
	}

	var vcs *tfeprovider.VCSRepo

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

		vcs = &tfeprovider.VCSRepo{
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

type TeamDataResource struct {
	ForEach      map[string]TeamDataResource `json:"for_each,omitempty"`
	Name         string                      `json:"name"`
	Organization string                      `json:"organization"`
}

// AppendTeamAccess adds the passed teams to the calling workspace
func AppendTeamAccess(module *tfconfig.Module, teamAccess TeamAccess, organization string) {
	if len(teamAccess) == 0 {
		return
	}

	dataForEach := map[string]TeamDataResource{}
	resourceForEach := map[string]tfeprovider.TeamAccess{}

	for _, access := range teamAccess {
		dataForEach[access.TeamName] = TeamDataResource{
			Name:         access.TeamName,
			Organization: organization,
		}

		teamIDRef := fmt.Sprintf("${data.tfe_team.teams[\"%s\"].id}", access.TeamName)

		resourceForEach[fmt.Sprintf("%s-%s", access.Workspace.Name, access.TeamName)] = tfeprovider.TeamAccess{
			TeamID:      teamIDRef,
			WorkspaceID: fmt.Sprintf("${tfe_workspace.workspace[%q].id}", access.Workspace.Name),
			Access:      access.Access,
			Permissions: access.ToResource().Permissions,
		}
	}

	if len(dataForEach) > 0 {
		module.AppendData("tfe_team", "teams", TeamDataResource{
			ForEach:      dataForEach,
			Name:         "${each.value.name}",
			Organization: "${each.value.organization}",
		})
	}

	module.AppendResource("tfe_team_access", "teams", tfeprovider.TeamAccess{
		ForEach:     resourceForEach,
		TeamID:      "${each.value.team_id}",
		WorkspaceID: "${each.value.workspace_id}",
		Access:      "${lookup(each.value, \"access\", null)}",
		DynamicPermissions: &tfeprovider.DynamicPermissions{
			Permission: []tfeprovider.DynamicPermissionEntry{{
				ForEach: "${lookup(each.value ,\"permissions\", null) != null ? {once: true} : {}}",
				Content: &tfeprovider.TeamAccessPermissions{
					Runs:             "${each.value.permissions.runs}",
					Variables:        "${each.value.permissions.variables}",
					StateVersions:    "${each.value.permissions.state_versions}",
					SentinelMocks:    "${each.value.permissions.sentinel_mocks}",
					WorkspaceLocking: "${each.value.permissions.workspace_locking}",
				},
			}},
		},
	})
}

type NewWorkspaceConfigOptions struct {
	Backend                  map[string]interface{}
	WorkspaceVariables       map[string]tfconfig.Variable
	RemoteStates             map[string]tfconfig.RemoteState
	Variables                Variables
	TeamAccess               TeamAccess
	WorkspaceResourceOptions *WorkspaceResourceOptions
	Providers                []Provider
}

// NewWorkspaceConfig takes in all required values for the Terraform workspace and outputs a struct that can be marshalled then planned or applied
func NewWorkspaceConfig(ctx context.Context, client *tfe.Client, workspaces []*Workspace, config *NewWorkspaceConfigOptions) (*tfconfig.Module, error) {
	wsResource, err := NewWorkspaceResource(ctx, client, workspaces, config.WorkspaceResourceOptions)
	if err != nil {
		return nil, err
	}

	module := &tfconfig.Module{
		Terraform: tfconfig.Terraform{},
		Variables: config.WorkspaceVariables,
		Data:      map[string]map[string]interface{}{},
		Resources: map[string]map[string]interface{}{
			"tfe_workspace": {
				"workspace": wsResource,
			},
		},
	}

	if config.Backend != nil {
		module.Terraform.Backend = config.Backend
	}

	for name, rs := range config.RemoteStates {
		module.AppendData("terraform_remote_state", name, rs)
	}

	for _, v := range config.Variables {
		module.AppendResource("tfe_variable", fmt.Sprintf("%s-%s", v.Workspace.Name, v.Key), v.ToResource())
	}

	AppendTeamAccess(module, config.TeamAccess, wsResource.Organization)

	AddProviders(module, config.Providers)

	return module, nil
}

func AddProviders(module *tfconfig.Module, providers []Provider) {
	if len(providers) == 0 {
		return
	}

	versions := map[string]tfconfig.RequiredProvider{}
	providerConfigs := map[string]tfconfig.ProviderConfig{}

	for _, p := range providers {
		versions[p.Name] = tfconfig.RequiredProvider{
			Source:  p.Source,
			Version: p.Version,
		}
		providerConfigs[p.Name] = p.Config
	}

	module.Providers = providerConfigs
	module.Terraform.RequiredProviders = versions
}

// WillDestroy parses a plan to look for whether the delete action is associated with any target resource
func WillDestroy(plan *tfjson.Plan, targetType string) bool {
	for _, rc := range plan.ResourceChanges {
		if rc.Type == targetType {
			for _, action := range rc.Change.Actions {
				if action == tfjson.ActionDelete {
					return true
				}
			}
		}
	}

	return false
}

// FindWorkspace returns a workspace that matches the passed Terraform workspace identifier (not the workspace name)
func FindWorkspace(workspaces []*Workspace, target string) *Workspace {
	for _, v := range workspaces {
		if v.Workspace == target {
			return v
		}
	}

	return nil
}
