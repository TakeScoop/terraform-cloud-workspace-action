package tfeprovider

type Config struct {
	Hostname string `json:"hostname"`
	Token    string `json:"token,omitempty"`
}
