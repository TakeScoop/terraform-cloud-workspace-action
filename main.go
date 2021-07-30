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

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/sethvargo/go-githubactions"
	yaml "gopkg.in/yaml.v2"

	"github.com/takescoop/terraform-cloud-workspace-action/internal/inputs"
)

func main() {
	token := githubactions.GetInput("terraform_token")
	host := githubactions.GetInput("terraform_host")
	name := strings.TrimSpace(githubactions.GetInput("name"))
	org := githubactions.GetInput("terraform_organization")

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", host),
		Token:   token,
	})
	if err != nil {
		log.Fatalf("error configuring Terraform client: %s", err)
	}

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
}`, host, token))

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(home, ".terraformrc"), b, 0644)
	if err != nil {
		log.Fatal(err)
	}

	workDir, err := ioutil.TempDir("", name)
	if err != nil {
		log.Fatal(err)
	}

	vcsType := githubactions.GetInput("vcs_type")
	vcsTokenID := githubactions.GetInput("vcs_token_id")

	var vcs *WorkspaceVCSBlock

	if vcsType != "" || vcsTokenID != "" {
		if vcsTokenID == "" {
			vcsTokenID, err = GetVCSTokenIDByClientType(context.Background(), client, org, vcsType)
			if err != nil {
				log.Fatalf("Failed to find VCS client: %s", err)
			}

			vcs = &WorkspaceVCSBlock{
				OauthTokenID:      vcsTokenID,
				Identifier:        githubactions.GetInput("vcs_repo"),
				IngressSubmodules: inputs.GetBool("vcs_ingress_submodules"),
			}
		}
	}

	b, err = json.MarshalIndent(WorkspaceConfig{
		Terraform: WorkspaceTerraform{
			Backend: WorkspaceBackend{
				S3: S3BackendConfig{},
			},
		},
		Variables: map[string]WorkspaceVariable{
			"organization": {
				Type: "string",
			},
			"terraform_version": {
				Type: "string",
			},
			"workspace_names": {
				Type: "set(string)",
			},
			"variables": {
				Type: "set(map(string))",
			},
		},
		Resources: map[string]map[string]interface{}{
			"tfe_workspace": {
				"workspace": WorkspaceWorkspaceResource{
					ForEach:          "${var.workspace_names}",
					Name:             "${each.value}",
					Organization:     "${var.organization}",
					AutoApply:        true,
					TerraformVersion: "${var.terraform_version}",
					VCSRepo:          vcs,
				},
			},
			"tfe_variable": {
				"variables": WorkspaceVariableResource{
					ForEach:     "${{ for k, v in var.variables : \"${v.workspace_name}-${v.key}\" => v }}",
					Description: "${each.value.description}",
					Key:         "${each.value.key}",
					Value:       "${each.value.value}",
					Category:    "${each.value.category}",
					WorkspaceID: "${tfe_workspace.workspace[each.value.workspace_name].id}",
				},
			},
		},
	}, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(workDir, "main.tf.json"), b, 0644)
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

	var workspaces []string

	if githubactions.GetInput("workspaces") == "" {
		workspaces = append(workspaces, name)
	} else {
		for _, ws := range strings.Split(githubactions.GetInput("workspaces"), ",") {
			workspaces = append(workspaces, fmt.Sprintf("%s-%s", name, strings.TrimSpace(ws)))
		}
	}

	genVars := []Variable{}
	err = yaml.Unmarshal([]byte(githubactions.GetInput("variables")), &genVars)
	if err != nil {
		log.Fatalf("Failed to parse variables %s", err)
	}

	wsVars := map[string][]Variable{}
	err = yaml.Unmarshal([]byte(githubactions.GetInput("workspace_variables")), &wsVars)
	if err != nil {
		log.Fatalf("Failed to parse workspace variables %s", err)
	}

	vars, err := ParseVariablesByWorkspace(workspaces, &genVars, &wsVars)
	if err != nil {
		log.Fatalf("Failed to parse variables: %s", err)
	}

	varBytes, err := json.Marshal(vars)
	if err != nil {
		log.Fatalf("Failed marshal vars: %s", err)
	}

	fmt.Println(string(varBytes))

	wsBytes, err := json.Marshal(workspaces)
	if err != nil {
		log.Fatalf("error marshalling workspaces input: %s", err)
	}

	varOpts := []*tfexec.VarOption{
		tfexec.Var(fmt.Sprintf("organization=%s", org)),
		tfexec.Var(fmt.Sprintf("terraform_version=%s", githubactions.GetInput("terraform_version"))),
		tfexec.Var(fmt.Sprintf("workspace_names=%s", string(wsBytes))),
		tfexec.Var(fmt.Sprintf("variables=%s", string(varBytes))),
	}

	if inputs.GetBool("import") {
		fmt.Println("Importing resources...")

		opts := make([]tfexec.ImportOption, len(varOpts))
		for i, v := range varOpts {
			opts[i] = v
		}

		for _, name := range workspaces {
			err = ImportWorkspace(context.Background(), tf, client, name, org, opts...)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	planPath := "plan.txt"

	var opts []tfexec.PlanOption
	for _, v := range varOpts {
		opts = append(opts, v)
	}

	opts = append(opts, tfexec.Out(planPath))

	diff, err := tf.Plan(
		context.Background(),
		opts...,
	)
	if err != nil {
		log.Fatalf("error running plan: %s", err)
	}

	if diff {
		planStr, err := tf.ShowPlanFileRaw(context.Background(), planPath)
		if err != nil {
			log.Fatalf("Error showing plan: %s", err)
		}

		fmt.Println(planStr)
		githubactions.SetOutput("plan", planStr)

		plan, err := tf.ShowPlanFile(context.Background(), planPath)
		if err != nil {
			log.Fatalf("error creating plan struct: %s", err)
		}

		b, err := json.Marshal(plan)
		if err != nil {
			log.Fatalf("error converting plan to json: %s", err)
		}

		githubactions.SetOutput("plan_json", string(b))

		if inputs.GetBool("apply") {
			fmt.Println("Applying...")
			err = tf.Apply(
				context.Background(),
				tfexec.DirOrPlan(planPath),
			)
			if err != nil {
				log.Fatalf("error running apply: %s", err)
			}
		}
	} else {
		fmt.Println("No changes")
	}
}
