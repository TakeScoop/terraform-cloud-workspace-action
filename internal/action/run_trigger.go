package action

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfconfig"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

type RunTriggerInput struct {
	SourceID   string `yaml:"id"`
	SourceName string `yaml:"name"`
}

type RunTriggerInputs []RunTriggerInput

type RunTrigger struct {
	SourceID     string
	Workspace    *Workspace
	WorkspaceRef map[string]tfeprovider.DataWorkspace
}

type RunTriggers []RunTrigger

func (rt *RunTriggerInput) ToRunTrigger(target *Workspace, workspaces []*Workspace, organization string) (*RunTrigger, error) {
	trigger := &RunTrigger{
		Workspace: target,
	}

	if rt.SourceID != "" && rt.SourceName != "" {
		return nil, fmt.Errorf("run trigger source ID and source name cannot both be set")
	}

	if rt.SourceID != "" {
		trigger.SourceID = rt.SourceID
	} else if rt.SourceName != "" {
		for _, ws := range workspaces {
			if ws.Name == rt.SourceName {
				trigger.SourceID = fmt.Sprintf("${tfe_workspace.workspace[%q].id}", ws.Workspace)
			}
		}

		if trigger.SourceID == "" {
			trigger.WorkspaceRef = map[string]tfeprovider.DataWorkspace{
				rt.SourceName: {
					Name:         rt.SourceName,
					Organization: organization,
				},
			}

			trigger.SourceID = fmt.Sprintf("${data.tfe_workspace.run_trigger_workspaces[%q].id}", rt.SourceName)
		}
	} else {
		return nil, fmt.Errorf("run trigger source ID or source name must be set")
	}

	return trigger, nil
}

// MergeRunTriggers returns a list of run triggers objects, one per workspace per trigger
func MergeRunTriggers(inputs RunTriggerInputs, workspaceInputs map[string]RunTriggerInputs, workspaces []*Workspace, organization string) (RunTriggers, error) {
	triggers := RunTriggers{}

	for _, input := range inputs {
		for _, ws := range workspaces {
			rt, err := input.ToRunTrigger(ws, workspaces, organization)
			if err != nil {
				return nil, err
			}

			triggers = append(triggers, *rt)
		}
	}

	for wsName, wsInputs := range workspaceInputs {
		ws := FindWorkspace(workspaces, wsName)
		if ws == nil {
			return nil, fmt.Errorf("failed to match workspace variable with known workspaces. Workspace %s not found", ws.Workspace)
		}

		for _, wsi := range wsInputs {
			rt, err := wsi.ToRunTrigger(ws, workspaces, organization)
			if err != nil {
				return nil, err
			}

			triggers = append(triggers, *rt)
		}
	}

	return triggers, nil
}

// ToResource returns a tfeprovider.RunTrigger object from the calling RunTrigger object
func (t RunTrigger) ToResource() *tfeprovider.RunTrigger {
	return &tfeprovider.RunTrigger{
		WorkspaceID:  fmt.Sprintf("${tfe_workspace.workspace[%q].id}", t.Workspace.Workspace),
		SourceableID: t.SourceID,
	}
}

// AppendRunTriggers takes a list of run triggers and adds them to the passed module
func AppendRunTriggers(module *tfconfig.Module, triggers RunTriggers) {
	if len(triggers) == 0 {
		return
	}

	triggerForEach := map[string]tfeprovider.RunTrigger{}

	wsDataForEach := map[string]tfeprovider.DataWorkspace{}

	for _, t := range triggers {
		if t.WorkspaceRef != nil {
			for name, ref := range t.WorkspaceRef {
				wsDataForEach[name] = ref
			}
		}

		triggerForEach[fmt.Sprintf("%s-%s", t.Workspace.Workspace, t.SourceID)] = *t.ToResource()
	}

	if len(wsDataForEach) > 0 {
		module.AppendData("tfe_workspace", "run_trigger_workspaces", tfeprovider.DataWorkspace{
			ForEach:      wsDataForEach,
			Name:         "${each.value.name}",
			Organization: "${each.value.organization}",
		})
	}

	rt := tfeprovider.RunTrigger{
		ForEach:      triggerForEach,
		SourceableID: "${each.value.sourceable_id}",
		WorkspaceID:  "${each.value.workspace_id}",
	}

	module.AppendResource("tfe_run_trigger", "trigger", rt)
}

// FetchInboundRunTriggers takes a workspace and returns related tfe.RunTrigger objects
func FetchInboundRunTriggers(ctx context.Context, client *tfe.Client, workspaceID string) ([]*tfe.RunTrigger, error) {
	rts, err := client.RunTriggers.List(ctx, workspaceID, tfe.RunTriggerListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: maxPageSize,
		},
		RunTriggerType: tfe.String("inbound"),
	})
	if err != nil {
		return nil, err
	}

	return rts.Items, nil
}

// ToRunTriggers takes a list of tfe.RunTriggers and returns a list of RunTriggers
func ToRunTriggers(tfeTriggers []*tfe.RunTrigger, workspace *Workspace) (runTriggers RunTriggers) {
	for _, rt := range tfeTriggers {
		runTriggers = append(runTriggers, RunTrigger{
			SourceID:  rt.Sourceable.ID,
			Workspace: workspace,
		})
	}

	return runTriggers
}
