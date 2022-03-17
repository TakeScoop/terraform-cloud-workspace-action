package action

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
)

func NewTerraformExec(ctx context.Context, workDir string, version string) (*tfexec.Terraform, error) {
	execPath, err := tfinstall.Find(
		ctx,
		tfinstall.ExactVersion(version, workDir),
	)
	if err != nil {
		return nil, err
	}

	return tfexec.NewTerraform(workDir, execPath)
}

func writeTerraformrcFile(host string, token string) error {
	b := []byte(fmt.Sprintf(`credentials %q { token = %q	}`, host, token))

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to retrieve homedir: %w", err)
	}

	err = ioutil.WriteFile(path.Join(home, ".terraformrc"), b, 0644)
	if err != nil {
		return fmt.Errorf("failed to write Terraform Cloud credentials to home directory: %w", err)
	}

	return nil
}
