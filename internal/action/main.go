package action

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
	"github.com/sethvargo/go-githubactions"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/action/inputs"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfconfig"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
	yaml "gopkg.in/yaml.v2"
)

func Run() {
	ctx := context.Background()

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

	workDir, err := ioutil.TempDir("", name)
	if err != nil {
		log.Fatalf("error creating temp dir: %s", err)
	}

	defer os.RemoveAll(workDir)

	tf, err := NewTerraformExec(ctx, workDir, githubactions.GetInput("runner_terraform_version"))
	if err != nil {
		log.Fatalf("error creating Terraform client: %s", err)
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

	var remoteStates map[string]tfconfig.RemoteState

	err = yaml.Unmarshal([]byte(githubactions.GetInput("remote_states")), &remoteStates)
	if err != nil {
		log.Fatalf("Failed to parse remote state blocks%s", err)
	}

	var workspaces []*Workspace

	if githubactions.GetInput("workspaces") == "" {
		workspaces = append(workspaces, &Workspace{
			Name:      name,
			Workspace: "default",
		})
	} else {
		var wsNames []string

		if err = yaml.Unmarshal([]byte(githubactions.GetInput("workspaces")), wsNames); err != nil {
			log.Fatalf("Failed to parse workspaces: %s", err)
		}

		for _, wsn := range wsNames {
			workspaces = append(workspaces, &Workspace{
				Name:      fmt.Sprintf("%s-%s", name, wsn),
				Workspace: wsn,
			})
		}
	}

	genVars := VariablesInput{}

	err = yaml.Unmarshal([]byte(githubactions.GetInput("variables")), &genVars)
	if err != nil {
		log.Fatalf("Failed to parse variables %s", err)
	}

	wsVars := WorkspaceVariablesInput{}

	err = yaml.Unmarshal([]byte(githubactions.GetInput("workspace_variables")), &wsVars)
	if err != nil {
		log.Fatalf("Failed to parse workspace variables %s", err)
	}

	wsNames := make([]string, len(workspaces))
	for i, ws := range workspaces {
		wsNames[i] = ws.Name
	}

	variables := Variables{}

	for _, ws := range workspaces {
		for _, v := range genVars {
			variables = append(variables, *NewVariable(v, ws))
		}
	}

	for wsName, wvs := range wsVars {
		ws := FindWorkspace(workspaces, wsName)

		if ws == nil {
			log.Fatalf("Failed to match workspace variable with known workspaces. Workspace %s not found", wsName)
		}

		for _, v := range wvs {
			variables = append(variables, *NewVariable(v, ws))
		}
	}

	var teamInputs TeamAccessInput

	if err = yaml.Unmarshal([]byte(githubactions.GetInput("team_access")), &teamInputs); err != nil {
		log.Fatalf("Failed to parse teams: %s", err)
	}

	for _, teamInput := range teamInputs {
		if err := teamInput.Validate(); err != nil {
			log.Fatal(err)
		}
	}

	teamAccess := NewTeamAccess(teamInputs, workspaces)

	backend, err := tfconfig.ParseBackend(githubactions.GetInput("backend_config"))
	if err != nil {
		log.Fatalf("Failed to parse backend: %s", err)
	}

	wsConfig, err := NewWorkspaceConfig(ctx, client, workspaces, &NewWorkspaceConfigOptions{
		Backend: backend,
		WorkspaceResourceOptions: &WorkspaceResourceOptions{
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
		},
		RemoteStates: remoteStates,
		Variables:    variables,
		TeamAccess:   teamAccess,
		Providers: []Provider{
			{
				Name:    "tfe",
				Version: githubactions.GetInput("tfe_provider_version"),
				Source:  "hashicorp/tfe",
				Config: tfeprovider.Config{
					Hostname: host,
					Token:    token,
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create new workspace configuration: %s", err)
	}

	b, err = json.MarshalIndent(wsConfig, "", "\t")
	if err != nil {
		log.Fatalf("Failed to marshal workspace configuration: %s", err)
	}

	if err = ioutil.WriteFile(path.Join(workDir, "main.tf.json"), b, 0644); err != nil {
		log.Fatal(err)
	}

	if err = tf.Init(ctx); err != nil {
		log.Fatalf("error running Init: %s", err)
	}

	if inputs.GetBool("import") || backend == nil {
		fmt.Println("Importing resources...")

		for _, ws := range workspaces {
			err = ImportWorkspace(ctx, tf, client, ws.Name, org)
			if err != nil {
				log.Fatal(err)
			}
		}

		for _, v := range variables {
			err = ImportVariable(ctx, tf, client, v.Key, v.Workspace.Name, org)
			if err != nil {
				log.Fatalf("Error importing variables: %s\n", err)
			}
		}

		for _, access := range teamAccess {
			if err = ImportTeamAccess(ctx, tf, client, org, access.Workspace.Name, access.TeamID); err != nil {
				log.Fatalf("Error importing team access: %s\n", err)
			}
		}
	}

	planPath := "plan.txt"

	opts := []tfexec.PlanOption{
		tfexec.Out(planPath),
	}

	diff, err := tf.Plan(ctx, opts...)
	if err != nil {
		log.Fatalf("error running plan: %s", err)
	}

	if diff {
		planStr, err := tf.ShowPlanFileRaw(ctx, planPath)
		if err != nil {
			log.Fatalf("Error showing plan: %s", err)
		}

		fmt.Println(planStr)
		githubactions.SetOutput("plan", planStr)

		plan, err := tf.ShowPlanFile(ctx, planPath)
		if err != nil {
			log.Fatalf("error creating plan struct: %s", err)
		}

		b, err := json.Marshal(plan)
		if err != nil {
			log.Fatalf("error converting plan to json: %s", err)
		}

		githubactions.SetOutput("plan_json", string(b))

		if !inputs.GetBool("allow_workspace_deletion") && WillDestroy(plan, "tfe_workspace") {
			log.Fatal("Error: Workspace planned for deletion. If this is intentional, set allow_workspace_deletion to true to proceed.")
		}

		if inputs.GetBool("apply") {
			fmt.Println("Applying...")

			if err = tf.Apply(ctx, tfexec.DirOrPlan(planPath)); err != nil {
				log.Fatalf("error running apply: %s", err)
			}
		}
	} else {
		fmt.Println("No changes")
	}
}
