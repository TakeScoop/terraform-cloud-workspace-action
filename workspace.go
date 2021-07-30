package main

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
)

type WorkspaceConfig struct {
	Terraform WorkspaceTerraform                `json:"terraform"`
	Variables map[string]WorkspaceVariable      `json:"variable,omitempty"`
	Resources map[string]map[string]interface{} `json:"resource,omitempty"`
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

type WorkspaceWorkspaceResource struct {
	ForEach          string             `json:"for_each,omitempty"`
	Name             string             `json:"name"`
	Organization     string             `json:"organization"`
	AutoApply        bool               `json:"auto_apply"`
	TerraformVersion string             `json:"terraform_version"`
	VCSRepo          *WorkspaceVCSBlock `json:"vcs_repo,omitempty"`
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

	return nil, fmt.Errorf("no VCS Client found named %s", vcsType)
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
