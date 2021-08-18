package action

import (
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfconfig"
)

type Provider struct {
	Version string
	Source  string
	Name    string
	Config  tfconfig.ProviderConfig
}
