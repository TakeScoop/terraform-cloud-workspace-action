package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	gotfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/sethvargo/go-githubactions"
	tfe "github.com/takescoop/terraform-cloud-workspace-action/generated/hashicorp/tfe"
	"github.com/takescoop/terraform-cloud-workspace-action/internal/action/inputs"
	"gopkg.in/yaml.v2"

	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

func Run() {
	ctx := context.Background()

	name := githubactions.GetInput("name")
	token := githubactions.GetInput("terraform_token")
	host := githubactions.GetInput("terraform_host")
	organization := githubactions.GetInput("terraform_organization")

	stackName := fmt.Sprintf("%s-stack", name)
	workDir := fmt.Sprintf("cdktf.out/stacks/%s", stackName)

	statefile := fmt.Sprintf("%s/terraform.tfstate", workDir)

	app := cdktf.NewApp(&cdktf.AppOptions{})

	stack := cdktf.NewTerraformStack(app, &stackName)

	cdktf.NewLocalBackend(stack, &cdktf.LocalBackendProps{
		Path: &statefile,
	})

	tfe.NewTfeProvider(stack, jsii.String("tfe"), &tfe.TfeProviderConfig{
		Token:    &token,
		Hostname: &host,
	})

	var workspaces []tfe.Workspace

	wsConfig := &tfe.WorkspaceConfig{
		Organization:     &organization,
		AgentPoolId:      jsii.String(githubactions.GetInput("agent_pool_id")),
		AllowDestroyPlan: jsii.Bool(inputs.GetBool("allow_destroy_plan")),
	}

	if githubactions.GetInput("workspaces") == "" {
		wsConfig.Name = &name
		workspaces = append(workspaces, tfe.NewWorkspace(stack, &name, wsConfig))
	} else {
		var wsInputs WorkspaceInputs
		if err := yaml.Unmarshal([]byte(githubactions.GetInput("workspaces")), wsInputs); err != nil {
			log.Fatalf("Failed to parse workspaces: %s", err)
		}

		for _, input := range wsInputs {
			newWSConfig := *wsConfig

			newWSConfig.Name = jsii.String(fmt.Sprintf("%s-%s", name, input))

			workspaces = append(workspaces, tfe.NewWorkspace(stack, &name, &newWSConfig))
		}
	}

	app.Synth()

	client, err := gotfe.NewClient(&gotfe.Config{
		Address: fmt.Sprintf("https://%s", host),
		Token:   token,
	})
	if err != nil {
		fmt.Println(token)
		log.Fatalf("error configuring Terraform client: %s", err)
	}

	execPath, err := tfinstall.Find(
		ctx,
		tfinstall.ExactVersion(githubactions.GetInput("runner_terraform_version"), "/tmp/"),
	)
	if err != nil {
		log.Fatalf("error installing terraform: %s", err)
	}

	tf, err := tfexec.NewTerraform(workDir, execPath)
	if err != nil {
		log.Fatalf("error creating Terraform client: %s", err)
	}

	if err = tf.Init(ctx); err != nil {
		log.Fatalf("error running Init: %s", err)
	}

	if inputs.GetBool("import") {
		fmt.Println("Importing resources...")

		for _, ws := range workspaces {
			err := ImportWorkspace(ctx, tf, client, *ws.NameInput(), *ws.Fqn(), organization)

			if err != nil {
				log.Fatal(err)
			}
		}
	}

	PlanAndApply(ctx, tf)
}

func PlanAndApply(ctx context.Context, tf *tfexec.Terraform) {
	planPath := "plan.txt"

	planOpts := []tfexec.PlanOption{
		tfexec.Out(planPath),
	}

	diff, err := tf.Plan(ctx, planOpts...)
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
