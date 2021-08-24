package tfconfig

type Terraform struct {
	Backend           map[string]interface{}      `json:"backend"`
	RequiredVersion   string                      `json:"required_version,omitempty"`
	RequiredProviders map[string]RequiredProvider `json:"required_providers,omitempty"`
}

type RequiredProvider struct {
	Source  string `json:"source,omitempty"`
	Version string `json:"version"`
}
