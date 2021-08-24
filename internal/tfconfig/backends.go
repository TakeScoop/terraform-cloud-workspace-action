package tfconfig

import (
	"encoding/json"

	yaml "sigs.k8s.io/yaml"
)

// Returns a generic backend object that can be directly added to the Terraform config, will return nil if no backend is set
func ParseBackend(backendInput string) (map[string]interface{}, error) {
	if backendInput == "" {
		return nil, nil
	}

	j, err := yaml.YAMLToJSON([]byte(backendInput))
	if err != nil {
		return nil, err
	}

	var backend map[string]interface{}

	if err = json.Unmarshal(j, &backend); err != nil {
		return nil, err
	}

	return backend, nil
}
