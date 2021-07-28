package inputs

import "github.com/sethvargo/go-githubactions"

func GetBool(name string) bool {
	return githubactions.GetInput(name) == "true"
}
