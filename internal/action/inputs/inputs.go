package inputs

import (
	"strings"

	"github.com/sethvargo/go-githubactions"
)

// GetBool returns true if the input value is "true", otherwise false
func GetBool(name string) bool {
	return strings.EqualFold(githubactions.GetInput(name), "true")
}

// GetBoolPtr returns nil if the value was unset, true if the input value is "true", otherwise false
func GetBoolPtr(name string) *bool {
	b := githubactions.GetInput(name)

	if b == "" {
		return nil
	}

	bp := strings.EqualFold(b, "true")

	return &bp
}
