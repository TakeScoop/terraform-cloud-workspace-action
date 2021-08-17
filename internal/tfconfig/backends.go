package tfconfig

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

type Backend struct {
	S3    *S3Backend    `yaml:"s3,omitempty" json:"s3,omitempty"`
	Local *LocalBackend `yaml:"s3,omitempty" json:"local,omitempty"`
}

// TODO: Support arbitrary backend types
// Avoid parsing into sepcific structs for specific backend types

type S3Backend struct {
	Bucket    string `yaml:"bucket" json:"bucket"`
	Key       string `yaml:"key" json:"key"`
	Region    string `yaml:"region" json:"region"`
	AccessKey string `yaml:"access_key" json:"access_key,omitempty"`
	SecretKey string `yaml:"secret_key" json:"secret_key,omitempty"`
	RoleArn   string `yaml:"role_arn" json:"role_arn,omitempty"`
}

type LocalBackend struct {
	Path string `json:"path,omitempty"`
}

func ParseBackend(backendInput string) (*Backend, error) {
	var backend map[string]interface{}

	wsBackend := &Backend{}

	if err := yaml.Unmarshal([]byte(backendInput), &backend); err != nil {
		return nil, err
	}

	if _, ok := backend["s3"]; ok {
		var s3Backend map[string]S3Backend

		if err := yaml.Unmarshal([]byte(backendInput), &s3Backend); err != nil {
			return nil, err
		}

		be := s3Backend["s3"]
		wsBackend.S3 = &be

		return wsBackend, nil
	}

	if _, ok := backend["local"]; ok {
		var localBackend map[string]LocalBackend

		if err := yaml.Unmarshal([]byte(backendInput), &localBackend); err != nil {
			return nil, err
		}

		be := localBackend["local"]
		wsBackend.Local = &be

		return wsBackend, nil
	}

	return nil, fmt.Errorf("unsupported backend type %v", backend)
}
