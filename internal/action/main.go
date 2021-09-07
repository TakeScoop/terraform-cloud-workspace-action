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
	apply := inputs.GetBool("apply")

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

	var wsInputs []string

	err = yaml.Unmarshal([]byte(githubactions.GetInput("workspaces")), &wsInputs)
	if err != nil {
		githubactions.Fatalf("Failed to decode workspaces: %s", err)
	}

	workspaces, err := ParseWorkspaces(wsInputs, name)
	if err != nil {
		githubactions.Fatalf("Failed to parse workspaces: %s", err)
	}

	if err := SetWorkspaceIDs(ctx, client, workspaces, org); err != nil {
		githubactions.Fatalf("Failed to set workspace IDs: %s", err)
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

	providers := []Provider{
		{
			Name:    "tfe",
			Version: githubactions.GetInput("tfe_provider_version"),
			Source:  "hashicorp/tfe",
			Config: tfeprovider.Config{
				Hostname: host,
				Token:    token,
			},
		},
	}

	module, err := NewWorkspaceConfig(ctx, client, workspaces, &NewWorkspaceConfigOptions{
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
		Providers:    providers,
	})
	if err != nil {
		githubactions.Fatalf("Failed to create new workspace configuration: %s", err)
	}

	filePath := path.Join(workDir, "main.tf.json")

	if err = TerraformInit(ctx, tf, module, filePath); err != nil {
		githubactions.Fatalf("Failed to initialize the Terraform configuration: %s", err)
	}

	if err = CopyStateToBackend(ctx, tf, module, nil, filePath); err != nil {
		githubactions.Fatalf("Failed to copy state to a local backend: %s", err)
	}

	if inputs.GetBool("import") {
		if err = ImportResources(ctx, client, tf, module, filePath, workspaces, org, providers); err != nil {
			githubactions.Fatalf("Failed to import resources: %s", err)
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
	} else {
		githubactions.Infof("No changes\n")
	}

	if apply {
		githubactions.Infof("Applying...\n")

		if err = CopyStateToBackend(ctx, tf, module, backend, filePath); err != nil {
			githubactions.Fatalf("Failed to copy local state to configured backend: %s", err)
		}

		if err = tf.Apply(ctx, tfexec.DirOrPlan(planPath)); err != nil {
			githubactions.Fatalf("Failed to apply: %s", err)
		}

		githubactions.Infof("Success\n")
	}
}
