package action

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
)

func TestCreateWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	envs := map[string]string{
		"terraform_token":          os.Getenv("tf_token"),
		"terrafrm_organization":    "ryanwholey",
		"terraform_host":           "app.terraform.io",
		"name":                     "action-test",
		"import":                   "true",
		"apply":                    "true",
		"tfe_provider_version":     "0.25.3",
		"runner_terraform_version": "1.0.5",
		"terraform_version":        "1.0.5",
	}

	for key, value := range envs {
		os.Setenv(fmt.Sprintf("INPUT_%s", strings.ToUpper(key)), value)
	}

	client, err := tfe.NewClient(&tfe.Config{
		Address: envs["terraform_host"],
		Token:   envs["terraform_token"],
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Workspaces.Read(ctx, envs["terraform_organization"], envs["name"])
	if err.Error() != "resource not found" {
		t.Fatal("test workspace already exists")
	}

	Run()

	ws, err := client.Workspaces.Read(ctx, envs["terraform_organization"], envs["name"])
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ws.Name, envs["name"])

	t.Cleanup(func() {
		for key := range envs {
			os.Unsetenv(fmt.Sprintf("INPUT_%s", strings.ToUpper(key)))
		}

		err = client.Workspaces.Delete(ctx, envs["terraform_organization"], envs["name"])
		if err != nil {
			t.Fatal(err)
		}
	})

}
