package tfconfig

type Module struct {
	Terraform Terraform                         `json:"terraform"`
	Variables map[string]Variable               `json:"variable,omitempty"`
	Resources map[string]map[string]interface{} `json:"resource,omitempty"`
	Data      map[string]map[string]interface{} `json:"data,omitempty"`
	Providers map[string]ProviderConfig         `json:"provider,omitempty"`
}

// AppendData appends a data source of type "sourceType" with name "name" to the workspace's data configuration
func (m *Module) AppendData(sourceType string, name string, source interface{}) {
	if m.Data == nil {
		m.Data = map[string]map[string]interface{}{}
	}

	if _, ok := m.Data[sourceType]; !ok {
		m.Data[sourceType] = map[string]interface{}{}
	}

	m.Data[sourceType][name] = source
}

// AppendData appends a resource of type "sourceType" with name "name" to the workspace's resources configuration
func (m *Module) AppendResource(sourceType string, name string, source interface{}) {
	if m.Resources == nil {
		m.Resources = map[string]map[string]interface{}{}
	}

	if _, ok := m.Resources[sourceType]; !ok {
		m.Resources[sourceType] = map[string]interface{}{}
	}

	m.Resources[sourceType][name] = source
}
