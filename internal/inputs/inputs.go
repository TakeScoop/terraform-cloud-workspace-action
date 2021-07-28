package inputs

import "github.com/sethvargo/go-githubactions"

func GetBool(name string) bool {
	return strings.EqualFold(githubactions.GetInput(name), "true")
}
