package action

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/sethvargo/go-githubactions"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfconfig"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
	yaml "gopkg.in/yaml.v2"
)

type Inputs struct {
	Token                  string
	Host                   string
	Name                   string
	Description            string
	Tags                   string
	WorkspaceTags          string
	Organization           string
	Apply                  bool
	RunnerTerraformVersion string
	RemoteStates           string
	Workspaces             string
	Variables              string
	WorkspaceVariables     string
	TeamAccess             string
	BackendConfig          string
	AgentPoolID            string
	AutoApply              *bool
	ExecutionMode          string
	FileTriggersEnabled    *bool
	GlobalRemoteState      *bool
	QueueAllRuns           *bool
	RemoteStateConsumerIDs string
	SpeculativeEnabled     *bool
	TerraformVersion       string
	RunTriggers            string
	WorkspaceRunTriggers   string
	SSHKeyID               string
	VCSIngressSubmodules   bool
	VCSRepo                string
	VCSTokenID             string
	VCSType                string
	WorkingDirectory       string
	TFEProviderVersion     string
	Import                 bool
	AllowWorkspaceDeletion bool
}

func Run(config *Inputs) error {
	ctx := context.Background()

	client, err := tfe.NewClient(&tfe.Config{
		Address: fmt.Sprintf("https://%s", config.Host),
		Token:   config.Token,
	})
	if err != nil {
		return fmt.Errorf("failed to create Terraform client: %w", err)
	}

	workDir, err := ioutil.TempDir("", config.Name)
	if err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	defer os.RemoveAll(workDir)

	tf, err := NewTerraformExec(ctx, workDir, config.RunnerTerraformVersion)
	if err != nil {
		return fmt.Errorf("failed to create tfexec instance: %w", err)
	}

	b := []byte(fmt.Sprintf(`credentials "%s" {
	token = "%s" 
}`, config.Host, config.Token))

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to retrieve homedir: %w", err)
	}

	err = ioutil.WriteFile(path.Join(home, ".terraformrc"), b, 0644)
	if err != nil {
		return fmt.Errorf("failed to write Terraform Cloud credentials to home directory: %w", err)
	}

	var remoteStates map[string]tfconfig.RemoteState

	err = yaml.Unmarshal([]byte(config.RemoteStates), &remoteStates)
	if err != nil {
		return fmt.Errorf("failed to parse remote state blocks: %w", err)
	}

	var wsInputs []string

	err = yaml.Unmarshal([]byte(config.Workspaces), &wsInputs)
	if err != nil {
		return fmt.Errorf("failed to decode workspaces: %w", err)
	}

	workspaces, err := ParseWorkspaces(wsInputs, config.Name)
	if err != nil {
		return fmt.Errorf("failed to parse workspaces: %w", err)
	}

	if err := SetWorkspaceIDs(ctx, client, workspaces, config.Organization); err != nil {
		return fmt.Errorf("failed to set workspace IDs: %w", err)
	}

	genVars := VariablesInput{}

	err = yaml.Unmarshal([]byte(config.Variables), &genVars)
	if err != nil {
		return fmt.Errorf("failed to parse variables %w", err)
	}

	wsVars := WorkspaceVariablesInput{}

	err = yaml.Unmarshal([]byte(config.WorkspaceVariables), &wsVars)
	if err != nil {
		return fmt.Errorf("failed to parse workspace variables %w", err)
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
			return fmt.Errorf("failed to match workspace variable with known workspaces. Workspace %s not found", wsName)
		}

		for _, v := range wvs {
			variables = append(variables, *NewVariable(v, ws))
		}
	}

	var teamInputs TeamAccessInput

	if err = yaml.Unmarshal([]byte(config.TeamAccess), &teamInputs); err != nil {
		return fmt.Errorf("failed to parse teams: %w", err)
	}

	teamAccess := NewTeamAccess(teamInputs, workspaces)

	backend, err := tfconfig.ParseBackend(config.BackendConfig)
	if err != nil {
		return fmt.Errorf("failed to parse backend configuration: %w", err)
	}

	var tagInputs Tags
	if err = yaml.Unmarshal([]byte(config.Tags), &tagInputs); err != nil {
		return fmt.Errorf("failed to decode tag names: %w", err)
	}

	var wsTagInputs map[string]Tags
	if err = yaml.Unmarshal([]byte(config.WorkspaceTags), &wsTagInputs); err != nil {
		return fmt.Errorf("failed to decode workspace tag names: %w", err)
	}

	tags, err := MergeWorkspaceTags(tagInputs, wsTagInputs, workspaces)
	if err != nil {
		return fmt.Errorf("failed to format workspace tags: %w", err)
	}

	var triggerInputs RunTriggerInputs
	if err = yaml.Unmarshal([]byte(config.RunTriggers), &triggerInputs); err != nil {
		return fmt.Errorf("failed to decode workspace tag names: %w", err)
	}

	var workspaceTriggerInputs map[string]RunTriggerInputs
	if err = yaml.Unmarshal([]byte(config.WorkspaceRunTriggers), &workspaceTriggerInputs); err != nil {
		return fmt.Errorf("failed to decode workspace tag names: %w", err)
	}

	triggers, err := MergeRunTriggers(triggerInputs, workspaceTriggerInputs, workspaces, config.Organization)
	if err != nil {
		return fmt.Errorf("failed to merge run triggers: %w", err)
	}

	providers := []Provider{
		{
			Name:    "tfe",
			Version: config.TFEProviderVersion,
			Source:  "hashicorp/tfe",
			Config: tfeprovider.Config{
				Hostname: config.Host,
				Token:    config.Token,
			},
		},
	}

	module, err := NewWorkspaceConfig(ctx, client, workspaces, &NewWorkspaceConfigOptions{
		Backend: backend,
		WorkspaceResourceOptions: &WorkspaceResourceOptions{
			AgentPoolID:            config.AgentPoolID,
			AutoApply:              config.AutoApply,
			Description:            config.Description,
			ExecutionMode:          config.ExecutionMode,
			FileTriggersEnabled:    config.FileTriggersEnabled,
			GlobalRemoteState:      config.GlobalRemoteState,
			Organization:           config.Organization,
			QueueAllRuns:           config.QueueAllRuns,
			RemoteStateConsumerIDs: config.RemoteStateConsumerIDs,
			SpeculativeEnabled:     config.SpeculativeEnabled,
			Tags:                   tags,
			TerraformVersion:       config.TerraformVersion,
			SSHKeyID:               config.SSHKeyID,
			VCSIngressSubmodules:   config.VCSIngressSubmodules,
			VCSRepo:                config.VCSRepo,
			VCSTokenID:             config.VCSTokenID,
			VCSType:                config.VCSType,
			WorkingDirectory:       config.WorkingDirectory,
		},
		RemoteStates: remoteStates,
		Variables:    variables,
		TeamAccess:   teamAccess,
		RunTriggers:  triggers,
		Providers:    providers,
	})
	if err != nil {
		return fmt.Errorf("failed to create new workspace configuration: %w", err)
	}

	filePath := path.Join(workDir, "main.tf.json")

	if err = TerraformInit(ctx, tf, module, filePath); err != nil {
		return fmt.Errorf("failed to initialize the Terraform configuration: %w", err)
	}

	if !config.Apply {
		// copy state to local backend to avoid mutating state when apply=false
		module.Terraform.Backend = nil

		if err = TerraformInit(ctx, tf, module, filePath); err != nil {
			return fmt.Errorf("failed to initialize the Terraform configuration: %w", err)
		}
	}

	if config.Import {
		if err = ImportResources(ctx, client, tf, module, filePath, workspaces, config.Organization, providers); err != nil {
			return fmt.Errorf("failed to import resources: %w", err)
		}
	}

	planPath := "plan.txt"

	planOpts := []tfexec.PlanOption{
		tfexec.Out(planPath),
	}

	diff, err := tf.Plan(ctx, planOpts...)
	if err != nil {
		return fmt.Errorf("failed to plan: %w", err)
	}

	if diff {
		planStr, err := tf.ShowPlanFileRaw(ctx, planPath)
		if err != nil {
			return fmt.Errorf("failed to show plan: %w", err)
		}

		githubactions.Infof(planStr)
		githubactions.SetOutput("plan", planStr)

		plan, err := tf.ShowPlanFile(ctx, planPath)
		if err != nil {
			return fmt.Errorf("failed to create plan struct: %w", err)
		}

		b, err := json.Marshal(plan)
		if err != nil {
			return fmt.Errorf("failed to convert plan to JSON: %w", err)
		}

		githubactions.SetOutput("plan_json", string(b))

		if !config.AllowWorkspaceDeletion && WillDestroy(plan, "tfe_workspace") {
			return fmt.Errorf("error: allow_workspace_deletion must be true to allow workspace deletion. Deleting a workspace will permanently, irrecoverably delete all of its stored Terraform state versions")
		}

		if config.Apply {
			githubactions.Infof("Applying...\n")

			if err = tf.Apply(ctx, tfexec.DirOrPlan(planPath)); err != nil {
				return fmt.Errorf("failed to apply: %w", err)
			}

			githubactions.Infof("Success\n")
		}
	} else {
		githubactions.Infof("No changes\n")
	}

	return nil
}
