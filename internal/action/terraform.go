package action

import (
	"context"

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
