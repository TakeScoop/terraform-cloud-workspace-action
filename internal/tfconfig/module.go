package tfconfig

type Module struct {
	Terraform Terraform                         `json:"terraform"`
	Variables map[string]Variable               `json:"variable,omitempty"`
	Resources map[string]map[string]interface{} `json:"resource,omitempty"`
	Data      map[string]map[string]interface{} `json:"data,omitempty"`
	Providers map[string]ProviderConfig         `json:"provider,omitempty"`
}

func (m *Module) AppendData(dataType string, sources map[string]interface{}) {
	if len(sources) == 0 {
		return
	}

	if _, ok := m.Data[dataType]; !ok {
		m.Data[dataType] = map[string]interface{}{}
	}

	for name, source := range sources {
		m.Data[dataType][name] = source
	}
}

type Resources map[string]interface{}

func (m *Module) AppendResource(resourceType string, resources Resources) {
	if len(resources) == 0 {
		return
	}

	if _, ok := m.Resources[resourceType]; !ok {
		m.Resources[resourceType] = map[string]interface{}{}
	}

	for name, source := range resources {
		m.Resources[resourceType][name] = source
	}
}
