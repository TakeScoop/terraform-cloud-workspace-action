package main

import (
	"context"
	"encoding/json"
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

	workDir, err := ioutil.TempDir("", githubactions.GetInput("name"))
	if err != nil {
		log.Fatal(err)
	}

	b = []byte(`
terraform {
	backend "s3" {}
}

variable "name" {}
variable "organization" {}
variable "terraform_version" {}

resource "tfe_workspace" "workspace" {
	name              = var.name
	organization      = var.organization
	auto_apply        = true
	terraform_version = var.terraform_version
}
`)

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

	diff, err := tf.Plan(
		context.Background(),
		tfexec.Var(fmt.Sprintf("name=%s", githubactions.GetInput("name"))),
		tfexec.Var(fmt.Sprintf("organization=%s", githubactions.GetInput("terraform_organization"))),
		tfexec.Var(fmt.Sprintf("terraform_version=%s", githubactions.GetInput("terraform_version"))),
		tfexec.Out("plan.txt"),
	)

	if err != nil {
		log.Fatalf("error running plan: %s", err)
	}

	if diff {
		planStr, err := tf.ShowPlanFileRaw(context.Background(), "plan.txt")
		if err != nil {
			log.Fatalf("Error showing plan: %s", err)
		}

		fmt.Println(planStr)
		githubactions.SetOutput("plan", planStr)

		plan, err := tf.ShowPlanFile(context.Background(), "plan.txt")
		if err != nil {
			log.Fatalf("error creating plan struct: %s", err)
		}

		b, err := json.Marshal(plan)
		if err != nil {
			log.Fatalf("error converting plan to json: %s", err)
		}

		fmt.Println(string(b))
		githubactions.SetOutput("plan_json", string(b))
	} else {
		fmt.Println("No changes")
	}

	err = tf.Apply(
		context.Background(),
		tfexec.Var(fmt.Sprintf("name=%s", githubactions.GetInput("name"))),
		tfexec.Var(fmt.Sprintf("organization=%s", githubactions.GetInput("terraform_organization"))),
		tfexec.Var(fmt.Sprintf("terraform_version=%s", githubactions.GetInput("terraform_version"))),
	)

	if err != nil {
		log.Fatalf("error running apply: %s", err)
	}
}
