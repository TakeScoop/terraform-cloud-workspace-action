package main

type Provider struct {
	Version string
	Source  string
	Name    string
	Config  ProviderConfig
}

type ProviderConfig interface{}

type TFEProvider struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token,omitempty"`
}
