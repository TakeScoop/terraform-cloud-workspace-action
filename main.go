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

	wsResource, err := NewWorkspaceResource(context.Background(), client, WorkspaceConfigOptions{
		AgentPoolID:            githubactions.GetInput("agent_pool_id"),
		AutoApply:              inputs.GetBoolPtr("auto_apply"),
		ExecutionMode:          githubactions.GetInput("execution_mode"),
		FileTriggersEnabled:    inputs.GetBoolPtr("file_triggers_enabled"),
		GlobalRemoteState:      inputs.GetBoolPtr("global_remote_state"),
		Organization:           org,
		QueueAllRuns:           inputs.GetBoolPtr("queue_all_runs"),
		RemoteStateConsumerIDs: githubactions.GetInput("remote_state_consumer_ids"),
		SpeculativeEnabled:     inputs.GetBoolPtr("speculative_enabled"),
		TerraformVersion:       githubactions.GetInput("terraform_version"),
		SSHKeyID:               githubactions.GetInput("ssh_key_id"),
		VCSIngressSubmodules:   inputs.GetBool("vcs_ingress_submodules"),
		VCSRepo:                githubactions.GetInput("vcs_repo"),
		VCSTokenID:             githubactions.GetInput("vcs_token_id"),
		VCSType:                githubactions.GetInput("vcs_type"),
		WorkingDirectory:       githubactions.GetInput("working_directory"),
	})
	if err != nil {
		log.Fatalf("Error structuring workspace resource: %s", err)
	}

	wsConfig := WorkspaceConfig{
		Terraform: WorkspaceTerraform{
			Backend: WorkspaceBackend{
				S3: S3BackendConfig{},
			},
		},
		Variables: map[string]WorkspaceVariable{
			"workspace_names": {
				Type: "set(string)",
			},
		},
		Data: map[string]map[string]interface{}{},
		Resources: map[string]map[string]interface{}{
			"tfe_workspace": {
				"workspace": wsResource,
			},
		},
	}

	var remoteStates map[string]RemoteStateBlock

	err = yaml.Unmarshal([]byte(githubactions.GetInput("remote_states")), &remoteStates)
	if err != nil {
		log.Fatalf("Failed to parse remote state blocks%s", err)
	}

	if len(remoteStates) > 0 {
		wsConfig.Data["terraform_remote_state"] = map[string]interface{}{}

		for name, block := range remoteStates {
			wsConfig.Data["terraform_remote_state"][name] = block
		}
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

	if len(vars) > 0 {
		varResources := map[string]interface{}{}

		for _, v := range vars {
			varResources[fmt.Sprintf("%s-%s", v.WorkspaceName, v.Key)] = NewWorkspaceVariableResource(&v)
		}

		wsConfig.Resources["tfe_variable"] = varResources
	}

	b, err = json.MarshalIndent(wsConfig, "", "\t")
	if err != nil {
		log.Fatalf("Failed marshal workspace configuration: %s", err)
	}

	if err = ioutil.WriteFile(path.Join(workDir, "main.tf.json"), b, 0644); err != nil {
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
		backendConfigs = append(backendConfigs, tfexec.BackendConfig(val))
	}

	if err = tf.Init(context.Background(), backendConfigs...); err != nil {
		log.Fatalf("error running Init: %s", err)
	}

	wsBytes, err := json.Marshal(workspaces)
	if err != nil {
		log.Fatalf("error marshalling workspaces input: %s", err)
	}

	varOpts := []*tfexec.VarOption{
		tfexec.Var(fmt.Sprintf("workspace_names=%s", string(wsBytes))),
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

		for _, v := range vars {
			err = ImportVariable(context.Background(), tf, client, v.Key, v.WorkspaceName, org, opts...)
			if err != nil {
				log.Fatalf("Error importing variables: %s\n", err)
			}
		}
	}

	planPath := "plan.txt"

	var opts []tfexec.PlanOption
	for _, v := range varOpts {
		opts = append(opts, v)
	}

	opts = append(opts, tfexec.Out(planPath))

	diff, err := tf.Plan(context.Background(), opts...)
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

			if err = tf.Apply(context.Background(), tfexec.DirOrPlan(planPath)); err != nil {
				log.Fatalf("error running apply: %s", err)
			}
		}
	} else {
		fmt.Println("No changes")
	}
}
