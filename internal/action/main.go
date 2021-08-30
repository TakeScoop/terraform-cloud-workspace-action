package action

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		githubactions.Fatalf("Failed to create Terraform client: %s", err)
	}

	workDir, err := ioutil.TempDir("", name)
	if err != nil {
		githubactions.Fatalf("Failed to create working directory: %s", err)
	}

	defer os.RemoveAll(workDir)

	tf, err := NewTerraformExec(ctx, workDir, githubactions.GetInput("runner_terraform_version"))
	if err != nil {
		githubactions.Fatalf("Failed to create tfexec instance: %s", err)
	}

	b := []byte(fmt.Sprintf(`credentials "%s" {
	token = "%s" 
}`, host, token))

	home, err := os.UserHomeDir()
	if err != nil {
		githubactions.Fatalf("Failed to retrieve homedir: %s", err)
	}

	err = ioutil.WriteFile(path.Join(home, ".terraformrc"), b, 0644)
	if err != nil {
		githubactions.Fatalf("Failed to write Terraform Cloud credentials to home directory: %s", err)
	}

	var remoteStates map[string]tfconfig.RemoteState

	err = yaml.Unmarshal([]byte(githubactions.GetInput("remote_states")), &remoteStates)
	if err != nil {
		githubactions.Fatalf("Failed to parse remote state blocks: %s", err)
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
			githubactions.Fatalf("Failed to parse workspaces: %s", err)
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
		githubactions.Fatalf("Failed to parse variables %s", err)
	}

	wsVars := WorkspaceVariablesInput{}

	err = yaml.Unmarshal([]byte(githubactions.GetInput("workspace_variables")), &wsVars)
	if err != nil {
		githubactions.Fatalf("Failed to parse workspace variables %s", err)
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
			githubactions.Fatalf("Failed to match workspace variable with known workspaces. Workspace %s not found", wsName)
		}

		for _, v := range wvs {
			variables = append(variables, *NewVariable(v, ws))
		}
	}

	var teamInputs TeamAccessInput

	if err = yaml.Unmarshal([]byte(githubactions.GetInput("team_access")), &teamInputs); err != nil {
		githubactions.Fatalf("Failed to parse teams: %s", err)
	}

	teamAccess := NewTeamAccess(teamInputs, workspaces)

	backend, err := tfconfig.ParseBackend(githubactions.GetInput("backend_config"))
	if err != nil {
		githubactions.Fatalf("Failed to parse backend configuration: %s", err)
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
			WorkingDirectory:       githubactions.GetInput("working_directory"),
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
		githubactions.Fatalf("Failed to create new workspace configuration: %s", err)
	}

	b, err = json.MarshalIndent(wsConfig, "", "\t")
	if err != nil {
		githubactions.Fatalf("Failed to marshal workspace configuration: %s", err)
	}

	if err = ioutil.WriteFile(path.Join(workDir, "main.tf.json"), b, 0644); err != nil {
		githubactions.Fatalf("Failed to write configuration to working directory: %s", err)
	}

	if err = tf.Init(ctx); err != nil {
		githubactions.Fatalf("Failed to run Init: %s", err)
	}

	if inputs.GetBool("import") || backend == nil {
		githubactions.Infof("Importing resources...\n")

		for _, ws := range workspaces {
			err = ImportWorkspace(ctx, tf, client, ws.Name, org)
			if err != nil {
				githubactions.Fatalf("Failed to import workspace: %s", err)
			}
		}

		for _, v := range variables {
			err = ImportVariable(ctx, tf, client, v.Key, v.Workspace.Name, org)
			if err != nil {
				githubactions.Fatalf("Failed to import variable: %s", err)
			}
		}

		for _, access := range teamAccess {
			if err = ImportTeamAccess(ctx, tf, client, org, access.Workspace.Name, access.TeamName); err != nil {
				githubactions.Fatalf("Failed to import team access: %s", err)
			}
		}
	}

	planPath := "plan.txt"

	planOpts := []tfexec.PlanOption{
		tfexec.Out(planPath),
	}

	diff, err := tf.Plan(ctx, planOpts...)
	if err != nil {
		githubactions.Fatalf("Failed to plan: %s", err)
	}

	if diff {
		planStr, err := tf.ShowPlanFileRaw(ctx, planPath)
		if err != nil {
			githubactions.Fatalf("Failed to show plan: %s", err)
		}

		githubactions.Infof(planStr)
		githubactions.SetOutput("plan", planStr)

		plan, err := tf.ShowPlanFile(ctx, planPath)
		if err != nil {
			githubactions.Fatalf("Failed to create plan struct: %s", err)
		}

		b, err := json.Marshal(plan)
		if err != nil {
			githubactions.Fatalf("Failed to convert plan to JSON: %s", err)
		}

		githubactions.SetOutput("plan_json", string(b))

		if !inputs.GetBool("allow_workspace_deletion") && WillDestroy(plan, "tfe_workspace") {
			githubactions.Fatalf("Error: allow_workspace_deletion must be true to allow workspace deletion. Deleting a workspace will permanently, irrecoverably delete all of its stored Terraform state versions.")
		}

		if inputs.GetBool("apply") {
			githubactions.Infof("Applying...\n")

			if err = tf.Apply(ctx, tfexec.DirOrPlan(planPath)); err != nil {
				githubactions.Fatalf("Failed to apply: %s", err)
			}
		}
	} else {
		githubactions.Infof("No changes")
	}
}
