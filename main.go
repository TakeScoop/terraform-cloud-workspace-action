package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

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

	bcfg := strings.Split(
		strings.TrimSpace(githubactions.GetInput("backend_config")),
		"\n",
	)

	var backendConfigs []tfexec.InitOption
	for _, val := range bcfg {
		backendConfigs = append(
			backendConfigs,
			tfexec.BackendConfig(val),
		)
	}

	err = tf.Init(
		context.Background(),
		backendConfigs...,
	)
	if err != nil {
		log.Fatalf("error running Init: %s", err)
	}

	diff, err := tf.Plan(
		context.Background(),
		tfexec.Out("plan.txt"),
		tfexec.Var(fmt.Sprintf("name=%s", githubactions.GetInput("name"))),
		tfexec.Var(fmt.Sprintf("organization=%s", githubactions.GetInput("terraform_organization"))),
		tfexec.Var(fmt.Sprintf("terraform_version=%s", githubactions.GetInput("terraform_version"))),
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

		fmt.Println("Applying...")
		err = tf.Apply(
			context.Background(),
			tfexec.Var(fmt.Sprintf("name=%s", githubactions.GetInput("name"))),
			tfexec.Var(fmt.Sprintf("organization=%s", githubactions.GetInput("terraform_organization"))),
			tfexec.Var(fmt.Sprintf("terraform_version=%s", githubactions.GetInput("terraform_version"))),
		)
		if err != nil {
			log.Fatalf("error running apply: %s", err)
		}
	} else {
		fmt.Println("No changes")
	}
}
