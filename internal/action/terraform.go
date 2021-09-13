package action

import (
	"context"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfconfig"
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

// CopyStateToBackend copies state from the current backend to the passed backend by running Terraform Init
func CopyStateToBackend(ctx context.Context, tf *tfexec.Terraform, module *tfconfig.Module, backend map[string]interface{}, filePath string) error {
	mod := module
	mod.Terraform.Backend = backend

	if err := TerraformInit(ctx, tf, mod, filePath); err != nil {
		return err
	}

	return nil
}
