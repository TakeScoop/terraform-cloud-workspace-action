package tfconfig

type Module struct {
	Terraform Terraform                         `json:"terraform"`
	Variables map[string]Variable               `json:"variable,omitempty"`
	Resources map[string]map[string]interface{} `json:"resource,omitempty"`
	Data      map[string]map[string]interface{} `json:"data,omitempty"`
	Providers map[string]ProviderConfig         `json:"provider,omitempty"`
}
