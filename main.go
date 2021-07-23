package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/sethvargo/go-githubactions"
)

func main() {
	tmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		log.Fatalf("error creating temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	execPath, err := tfinstall.Find(
		context.Background(),
		tfinstall.ExactVersion(githubactions.GetInput("runner_terraform_version"), tmpDir),
	)

	if err != nil {
		log.Fatalf("error locating Terraform binary: %s", err)
	}

	b := []byte(fmt.Sprintf(`credentials "%s" {
	token = "%s" 
}`, githubactions.GetInput("terraform_host"), githubactions.GetInput("terraform_token")))

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(home, ".terraformrc"), b, 0644)
	if err != nil {
		log.Fatal(err)
	}

	workDir := path.Join(os.Getenv("GITHUB_WORKSPACE"), "tmp", githubactions.GetInput("name"))
	err = os.Mkdir(workDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	b = []byte(fmt.Sprintf(`
terraform {
	backend "s3" {}
}

resource "tfe_workspace" "workspace" {
	name         = "%s"
	organization = "%s"
	auto_apply = true
	terraform_version = "%s"
}
`,
		githubactions.GetInput("terraform_organization"),
		githubactions.GetInput("name"),
		githubactions.GetInput("terraform_version"),
	))

	err = ioutil.WriteFile(path.Join(workDir, "main.tf"), b, 0644)
	if err != nil {
		log.Fatal(err)
	}

	tf, err := tfexec.NewTerraform(workDir, execPath)
	if err != nil {
		log.Fatalf("error creating Terraform client: %s", err)
	}

	err = tf.Init(
		context.Background(),
		tfexec.BackendConfig(fmt.Sprintf("access_key=%s", githubactions.GetInput("aws_access_key"))),
		tfexec.BackendConfig(fmt.Sprintf("secret_key=%s", githubactions.GetInput("aws_secret_key"))),
		tfexec.BackendConfig(fmt.Sprintf("role_arn=%s", githubactions.GetInput("aws_role"))),
		tfexec.BackendConfig(fmt.Sprintf("bucket=%s", githubactions.GetInput("aws_storage_bucket"))),
		tfexec.BackendConfig(fmt.Sprintf("key=%s/terraform.tfstate", githubactions.GetInput("name"))),
		tfexec.BackendConfig(fmt.Sprintf("region=%s", githubactions.GetInput("aws_region"))),
	)
	if err != nil {
		log.Fatalf("error running Init: %s", err)
	}

	diff, err := tf.Plan(context.Background())
	if err != nil {
		log.Fatalf("error running plan: %s", err)
	}

	if diff {
		fmt.Println("Plan is not empty")
	}

	err = tf.Apply(context.Background())
	if err != nil {
		log.Fatalf("error running apply: %s", err)
	}
}
