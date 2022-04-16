package action

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
)

func NewTerraformExec(ctx context.Context, workDir string, tfVersion string) (*tfexec.Terraform, error) {
	v, err := version.NewVersion(tfVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Terraform version: %w", err)
	}

	installer := install.NewInstaller()
	execPath, err := installer.Ensure(ctx, []src.Source{
		&releases.ExactVersion{
			Product: product.Terraform,
			Version: v,
		},
	})

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
