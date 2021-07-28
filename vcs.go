package main

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
)

// FormatVCSBlock formats a vcs_repo block
func FormatVCSBlock(VCStokenID string, repo string, ingressSubmodules bool) string {
	return fmt.Sprintf(`
  vcs_repo {
    oauth_token_id     = %q
    identifier         = %q
		ingress_submodules = %t
  }
`, VCStokenID, repo, ingressSubmodules)
}

// getVCSClientByName looks for a VCS client of the passed name against the VCS clients in the Terraform Cloud organization
func getVCSClientByName(ctx context.Context, tfc *tfe.Client, organization string, vcsName string) (*tfe.OAuthClient, error) {
	list, err := tfc.OAuthClients.List(ctx, organization, tfe.OAuthClientListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, v := range list.Items {
		if v.ServiceProviderName == vcsName {
			return v, nil
		}
	}

	return nil, fmt.Errorf("no VCS Client found named %s", vcsName)
}

// GetVCSTokenIDByClientName returns an OAuth client token ID for the passed VCS client name
func GetVCSTokenIDByClientName(ctx context.Context, tfc *tfe.Client, organization string, vcsName string) (string, error) {
	vcsClient, err := getVCSClientByName(ctx, tfc, organization, vcsName)
	if err != nil {
		return "", err
	}

	if len(vcsClient.OAuthTokens) == 0 {
		return "", fmt.Errorf("no VCS tokens found for client %s:%s", vcsClient.ServiceProviderName, vcsClient.ID)
	}

	return vcsClient.OAuthTokens[0].ID, nil
}