package action

import (
	"fmt"

	"github.com/takescoop/terraform-cloud-workspace-action/internal/tfeprovider"
)

type RunTriggerInput string

type RunTriggerInputs []RunTriggerInput

type RunTrigger struct {
	SourceID  string
	Workspace *Workspace
}

type RunTriggers []RunTrigger

// MergeRunTriggers returns a list of run triggers objects, one per workspace per trigger
func MergeRunTriggers(inputs RunTriggerInputs, workspaceInputs map[string]RunTriggerInputs, workspaces []*Workspace) (RunTriggers, error) {
	triggers := RunTriggers{}

	for _, input := range inputs {
		for _, ws := range workspaces {
			triggers = append(triggers, RunTrigger{
				Workspace: ws,
				SourceID:  string(input),
			})
		}
	}

	for ws, wsInputs := range workspaceInputs {
		w := FindWorkspace(workspaces, ws)
		if w == nil {
			return nil, fmt.Errorf("failed to match workspace variable with known workspaces. Workspace %s not found", ws)
		}

		for _, wsi := range wsInputs {
			triggers = append(triggers, RunTrigger{
				Workspace: w,
				SourceID:  string(wsi),
			})
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
