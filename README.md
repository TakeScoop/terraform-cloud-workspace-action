# Terraform Cloud Workspace Action

A GitHub action for managing Terraform Cloud workspaces

## Usage

```yaml
name: TFLint
on: [push]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: takescoop/terraform-cloud-workspace-action@v0
        with:
          terraform_token: "${{ secrets.TF_TOKEN }}"
          terraform_organization: "my-org"
          apply: "${{ github.ref == format('refs/heads/{0}', github.event.repository.default_branch) }}"
          backend_config: |-
            s3:
              bucket: my-bucket
              key: foo.tfstate
              region: us-east-1
```

## Inputs

| Name | Description | Default |
| --- | --- | --- |
| `allow_workspace_deletion` | Whether to allow workspaces to be deleted. If enabled, workspace state may be irrecoverably deleted. | `false` |
| `apply` | (required) Whether to apply the proposed Terraform changes | |
| `terraform_organization` | (required) Terraform Cloud organization | |
| `terraform_token`  | (required) Terraform Cloud token | |
| `agent_pool_id` | ID of an agent pool to assign to the workspace. If passed, execution_mode is set to "agent" | |
| `auto_apply` | Whether to set auto_apply on the workspace or workspaces | true |
| `backend_config` | YAML encoded backend configurations | |
| `execution_mode` | Execution mode to use for the workspace | |
| `file_triggers_enabled` | Whether to filter runs based on the changed files in a VCS push | |
| `global_remote_state` | Whether all workspaces in the organization can access the workspace via remote state | `false` |
| `import` | Whether to attempt to import existing matching resources using the resource name | `false` |
| `name` | Name of the workspace. Becomes a prefix if workspaces are passed (`${name}-${workspace}`) | `"${{ github.event.repository.name }}" `|
| `queue_all_runs` | Whether the workspace should start automatically performing runs immediately after creation | |
| `remote_state_consumer_ids` | Comma separated list | default `""`
| `remote_states` | YAML encoded remote state blocks to configure in the workspace | |
| `runner_terraform_version` | Terraform version used to create the workspace | `1.0.3` |
| `speculative_enabled` | Whether the workspace allows speculative plans | |
| `ssh_key_id` | SSH key ID to assign the workspace | |
| `team_access` | YAML encoded teams and their associated permissions to be granted to the created workspaces | `false` |
| `terraform_version` | Terraform version | `1.0.3` |
| `terraform_host` | Terraform Cloud host | `app.terraform.io` |
| `tfe_provider_version` | Terraform Cloud provider version | `0.25.3` |
| `variables` | YAML encoded variables to apply to all workspaces | `""`
| `vcs_ingress_submodules` | Whether to allow submodule ingress | `false` |
| `vcs_repo` | Repository identifier for a VCS integration. Required if `vcs_name` or `vcs_token_id` are passed | `"${{ github.repository }}"` |
| `vcs_token_id` | Terraform VCS client token ID. Takes precedence over `vcs_name`. If neither are passed, no VCS integration is added. | |
| `vcs_type` | Terraform VCS type (e.g., "github"). Superseded by `vcs_token_id`. If neither are passed, no VCS integration is added | |
| `working_directory` | A relative path that Terraform will execute within. Defaults to the root of your repository | |
| `workspace_variables` | YAML encoded variables to apply to specific workspaces, with variables nested under workspace names | `""` |
| `workspaces` | YAML encoded list of workspace names | |

### Backend Config

This project supports two backend types, `S3` and `local`

```yml
with:
  ...
  backend_config: |-
    s3:
      bucket: my-bucket
      key: foo.tfstate
      region: us-east-1
      role_arn: arn:aws:iam::123456789:role/terraform
      access_key: xxx
      secret_key: xxx
```

### Variables and Workspace Variables

Variables are applied to all created workspaces, where workspace variables are applied to the noted workspace

```yml
...
with:
  workspaces: |-
    - staging
    - production
  variables: |-
    - key: general-secret
      value: "${{ secrets.SECRET }}"
      sensitive: true
  workspace_variables: |-
    staging:
      - key: environment
        value: staging
    production:
      - key: environment
        value: production
```

### Remote States

Remote states can be configured and referenced from other input fields

```yml
...
with:
  variables: |-
    - key: s3_secret
      value: ${data.terraform_remote_state.workspace_s3.outputs.secret}
    - key: tf_cloud_secret
      value: ${data.terraform_remote_state.workspace_tf_cloud.outputs.secret}
  remote_states: |-
    workspace_s3:
      backend: remote
      config:
        bucket: s3-bucket
        key: terraform.tfstate
        region: us-east-1
    workspace_tf_cloud:
      backend: remote
      config:
        hostname: app.terraform.io
        organization: organization
        workspaces:
          name: workspace-tf-cloud
```

### Team access

Create or update existing team access resources. Team `id` and `name` cannot both be simultaneously set.

```yml
with:
  team_access: |-
    - name: Readers
      access: read
    - id: team-abc123
      access: write
    - name: ${data.terraform_remote_state.tfe.outputs.teams["Engineering"].name}
      permissions:
        runs: read
        variables: read
        state_versions: read
        sentinel_mocks: read
        workspace_locking: true
  remote_states: |-
    tfe:
      backend: remote
      config:
        bucket: s3-bucket
        key: terraform.tfstate
        region: us-east-1
```

To import existing team access resources, a static value for `team_id` must be supplied

```yml
with:
  import: true
  team_access: |-
    - id: team-abc123
      access: write
    - id: ${data.terraform_remote_state.tfe.outputs.teams["Engineering"].id} # this will error
      access: read
      
```

### Importing existing resources

Set `import` to `true` for the action to attempt to import existing resources of matching values within the Terraform Cloud organization

```yml
...
with:
  import: true
```

## Outputs

| Name | Description |
| --- | --- |
| `plan` | A human friendly output of the Terraform plan |
| `plan_json` | A JSON representation of the Terraform plan |
