package tfeprovider

type Config struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token,omitempty"`
}

type ForEach struct {
	ForEach string `json:"for_each,omitempty"`
}
