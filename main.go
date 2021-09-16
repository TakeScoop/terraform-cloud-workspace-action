package main

import (
	"strings"

	"github.com/sethvargo/go-githubactions"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/action"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/action/inputs"
)

func main() {
	if err := action.Run(&action.Inputs{
		Token:                  githubactions.GetInput("terraform_token"),
		Host:                   githubactions.GetInput("terraform_host"),
		Name:                   strings.TrimSpace(githubactions.GetInput("name")),
		Description:            githubactions.GetInput("description"),
		Tags:                   githubactions.GetInput("tags"),
		WorkspaceTags:          githubactions.GetInput("workspace_tags"),
		Organization:           githubactions.GetInput("terraform_organization"),
		Apply:                  inputs.GetBool("apply"),
		RunnerTerraformVersion: githubactions.GetInput("runner_terraform_version"),
		RemoteStates:           githubactions.GetInput("remote_states"),
		Workspaces:             githubactions.GetInput("workspaces"),
		Variables:              githubactions.GetInput("variables"),
		WorkspaceVariables:     githubactions.GetInput("workspace_variables"),
		TeamAccess:             githubactions.GetInput("team_access"),
		BackendConfig:          githubactions.GetInput("backend_config"),
		AgentPoolID:            githubactions.GetInput("agent_pool_id"),
		AutoApply:              inputs.GetBoolPtr("auto_apply"),
		ExecutionMode:          githubactions.GetInput("execution_mode"),
		FileTriggersEnabled:    inputs.GetBoolPtr("file_triggers_enabled"),
		GlobalRemoteState:      inputs.GetBoolPtr("global_remote_state"),
		QueueAllRuns:           inputs.GetBoolPtr("queue_all_runs"),
		RemoteStateConsumerIDs: githubactions.GetInput("remote_state_consumer_ids"),
		SpeculativeEnabled:     inputs.GetBoolPtr("speculative_enabled"),
		TerraformVersion:       githubactions.GetInput("terraform_version"),
		RunTriggers:            githubactions.GetInput("run_triggers"),
		WorkspaceRunTriggers:   githubactions.GetInput("workspace_run_triggers"),
		SSHKeyID:               githubactions.GetInput("ssh_key_id"),
		VCSIngressSubmodules:   inputs.GetBool("vcs_ingress_submodules"),
		VCSRepo:                githubactions.GetInput("vcs_repo"),
		VCSTokenID:             githubactions.GetInput("vcs_token_id"),
		VCSType:                githubactions.GetInput("vcs_type"),
		WorkingDirectory:       githubactions.GetInput("working_directory"),
		TFEProviderVersion:     githubactions.GetInput("tfe_provider_version"),
		Import:                 inputs.GetBool("import"),
		AllowWorkspaceDeletion: inputs.GetBool("allow_workspace_deletion"),
	}); err != nil {
		githubactions.Fatalf("Error: %s", err)
	}
}
