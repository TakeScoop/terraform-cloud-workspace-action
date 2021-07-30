package inputs

import (
	"strings"

	"github.com/sethvargo/go-githubactions"
)

// GetBool returns true if the input value is "true", otherwise false
func GetBool(name string) bool {
	return strings.EqualFold(githubactions.GetInput(name), "true")
}

// GetStringPtr return nil if the is unset, otherwise returns a pointer to the value set at the input string
func GetStringPtr(name string) *string {
	s := githubactions.GetInput(name)

	if s == "" {
		return nil
	}

	return &s
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
