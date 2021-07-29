package main

import (
	"github.com/sethvargo/go-githubactions"
	yaml "gopkg.in/yaml.v2"
)

type customVariable struct {
	key         string `yaml:"key"`
	value       string `yaml:"value"`
	description string `yaml:"description,omitempty" default:""`
	category    string `yaml:"category,omitempty" default:"env"`
	sensitive   bool   `yaml:"sensitive,omitempty" default:true`
}

func ParseVariableInput() {
	vars := githubactions.GetInput("variables")

	yaml.Unmarshal([]bytes(`foo`))
}
