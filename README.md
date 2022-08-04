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

<!-- https://github.com/actions-ecosystem/describe-action -->

|             NAME             |                                                                               DESCRIPTION                                                                               | REQUIRED |                   DEFAULT                    |
|------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|----------------------------------------------|
| `agent_pool_id`              | ID of an agent pool to assign to the workspace. If passed, execution_mode is set to "agent".                                                                            | `false`  | `N/A`                                        |
| `allow_workspace_deletion`   | Whether to allow workspaces to be deleted. If enabled, workspace state may be irrecoverably deleted.                                                                    | `false`  | `false`                                      |
| `apply`                      | Whether to apply the proposed Terraform changes.                                                                                                                        | `true`   | `N/A`                                        |
| `auto_apply`                 | Whether to set auto_apply on the workspace or workspaces.                                                                                                               | `false`  | `true`                                       |
| `backend_config`             | YAML encoded backend configurations.                                                                                                                                    | `false`  | `N/A`                                        |
| `description`                | Terraform Cloud workspace description                                                                                                                                   | `false`  | `${{ github.event.repository.description }}` |
| `execution_mode`             | Execution mode to use for the workspace.                                                                                                                                | `false`  | `remote`                                     |
| `file_triggers_enabled`      | Whether to filter runs based on the changed files in a VCS push.                                                                                                        | `false`  | `N/A`                                        |
| `global_remote_state`        | Whether all workspaces in the organization can access the workspace via remote state.                                                                                   | `false`  | `false`                                      |
| `import`                     | Whether to import existing matching resources from the Terraform Cloud organization.                                                                                    | `false`  | `true`                                       |
| `name`                       | Name of the workspace. Becomes a prefix if workspaces are passed (`${name}-${workspace}`).                                                                              | `false`  | `${{ github.event.repository.name }}`        |
| `notification_configuration` | A YAML encoded map of notification settings applied to all created workspaces                                                                                           | `false`  | `N/A`                                        |
| `queue_all_runs`             | Whether the workspace should start automatically performing runs immediately after creation.                                                                            | `false`  | `N/A`                                        |
| `remote_state_consumer_ids`  | Comma separated list of workspace IDs to allow read access to the workspace outputs.                                                                                    | `false`  | `N/A`                                        |
| `remote_states`              | YAML encoded remote state blocks to configure in the workspace.                                                                                                         | `false`  | `N/A`                                        |
| `run_triggers`               | YAML encoded list of either workspace IDs or names that, when applied, trigger runs in all the created workspaces (max 20)                                              | `false`  | `N/A`                                        |
| `runner_terraform_version`   | Terraform version used to create the workspace.                                                                                                                         | `false`  | `1.1.8`                                      |
| `speculative_enabled`        | Whether the workspace allows speculative plans.                                                                                                                         | `false`  | `N/A`                                        |
| `ssh_key_id`                 | SSH key ID to assign the workspace.                                                                                                                                     | `false`  | `N/A`                                        |
| `tags`                       | YAML encoded list of tag names applied to all workspaces                                                                                                                | `false`  | `N/A`                                        |
| `team_access`                | YAML encoded teams and their associated permissions to be granted to the created workspaces.                                                                            | `false`  | `N/A`                                        |
| `terraform_host`             | Terraform Cloud host.                                                                                                                                                   | `false`  | `app.terraform.io`                           |
| `terraform_organization`     | Terraform Cloud organization.                                                                                                                                           | `true`   | `N/A`                                        |
| `terraform_token`            | Terraform Cloud token.                                                                                                                                                  | `true`   | `N/A`                                        |
| `terraform_version`          | Workspace Terraform version. This can be either an exact version or a version constraint (like ~> 1.0.0).                                                               | `false`  | `1`                                          |
| `tfe_provider_version`       | Terraform Cloud provider version.                                                                                                                                       | `false`  | `0.30.2`                                     |
| `variables`                  | YAML encoded variables to apply to all workspaces.                                                                                                                      | `false`  | `N/A`                                        |
| `vcs_ingress_submodules`     | Whether to allow submodule ingress.                                                                                                                                     | `false`  | `false`                                      |
| `vcs_repo`                   | Repository identifier for a VCS integration.                                                                                                                            | `false`  | `${{ github.repository }}`                   |
| `vcs_token_id`               | Terraform VCS client token ID. Takes precedence over `vcs_name`. If neither are passed, no VCS integration is added.                                                    | `false`  | `N/A`                                        |
| `vcs_type`                   | Terraform VCS type (e.g., "github"). Superseded by `vcs_token_id`. If neither are passed, no VCS integration is added.                                                  | `false`  | `N/A`                                        |
| `working_directory`          | A relative path that Terraform will execute within. Defaults to the root of your repository.                                                                            | `false`  | `N/A`                                        |
| `workspace_run_triggers`     | A YAML encoded map of workspaces to workspace IDs or names, which like `run_triggers`, will trigger a run for the associated workspace when the source workspace is ran | `false`  | `N/A`                                        |
| `workspace_tags`             | YAML encoded map of workspace names to a list of tag names, which are applied to the specified workspace                                                                | `false`  | `N/A`                                        |
| `workspace_variables`        | YAML encoded variables to apply to specific workspaces, with variables nested under workspace names.                                                                    | `false`  | `N/A`                                        |
| `workspaces`                 | YAML encoded list of workspace names.                                                                                                                                   | `false`  | `N/A`                                        |


### Backend Config

This project supports any backend supported by the selected Terraform version. The backend is used to persist the state of the Terraform Cloud workspace itself and its related resources (e.g., variables, teams). You generally should not pass "remote" workspace configuration, since that creates a circular dependency. 

If no backend is passed, the default Terraform local backend will be used.
**NOTE** When using the default local backend, `import` should always be `true` to ensure that resources can be managed across action runs. 

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

`variables` are applied to all created workspaces, where `workspace_variables` are applied to the noted workspace. Per the [workspace docs](https://www.terraform.io/docs/cloud/workspaces/variables.html), `category` field must be set to either `env` or `terraform`.

```yml
...
with:
  workspaces: |-
    - staging
    - production
  variables: |-
    - key: general-secret
      value: "${{ secrets.SECRET }}"
      category: env
      sensitive: true
  workspace_variables: |-
    staging:
      - key: environment
        value: staging
        category: terraform
    production:
      - key: environment
        value: production
        category: terraform
```

#### Remote state variable reference

Remote states can be configured and referenced for the variable `value` field

```yml
...
with:
  variables: |-
    - key: s3_secret
      value: ${data.terraform_remote_state.workspace_s3.outputs.secret}
      category: env
    - key: tf_cloud_secret
      value: ${data.terraform_remote_state.workspace_tf_cloud.outputs.secret}
      category: env
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
    - name: Writers
      permissions:
        runs: apply
        variables: write
        state_versions: write
        sentinel_mocks: read
        workspace_locking: true
```

### Importing existing resources

By default, the action will import any existing resources it can find based on a unique attribute. It makes multiple passes to discover all existing resources, first finding matching workspaces and then related resources (variables, team access).

When `apply` is set to `false`, the configured backend state will be copied to a local backend and `import` will be set to `true`. This grants some visibility into the import changes before they are actually applied to the configured backend.

To disable the import feature, set `import` to `false`

```yml
...
with:
  import: false
```

### Workspace tags

Workspace tags can be specified in two ways, `tags` and `workspace_tags`. `tags` apply to every workspace, while `workspace_tags` apply to the specified workspace only 

```yml
tags: |-
- all
workspace_tags: |-
  staging:
    - staging
  production:
    - production
```

### Run Triggers

The following configuration will add a run trigger for the `alpha` and `beta` workspaces when workspace `parent-workspace` is ran, and will also add two more triggers to the `alpha` workspace when either workspace `ws-abc123` or `ws-def456` are ran

```yml
workspaces: |-
  - alpha
  - beta
run_triggers: |-
  - name: parent-workspace
workspace_run_triggers: |-
  alpha:
    - id: ws-abc123
    - id: ws-def456
```

### Notification configuration

The following configuration will add a [notification configuration](https://registry.terraform.io/providers/hashicorp/tfe/latest/docs/resources/notification_configuration#destination_type) for each workspace. 

```yml
workspaces: |-
  - alpha
  - beta
notification_configuration: |-
  name: my-notification
  destination_type: email
  email_addresses:
    - foo@email.com
  enabled: true
```

## Outputs

| Name | Description |
| --- | --- |
| `plan` | A human friendly output of the Terraform plan |
| `plan_json` | A JSON representation of the Terraform plan |

## Development

### Test

To test the project

`go test -v -short ./...`

### Lint

This project uses [`golangci-lint`](https://github.com/golangci/golangci-lint)

To lint the project

`golangci-lint run ./...`

To auto fix issues where supported

`golangci-lint run  --fix ./...`
