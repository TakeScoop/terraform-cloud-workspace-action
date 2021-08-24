package tfconfig

import (
	"encoding/json"

	yaml "sigs.k8s.io/yaml"
)

func ParseBackend(backendInput string) (map[string]interface{}, error) {
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
