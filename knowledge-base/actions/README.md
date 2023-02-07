# Contribute to the GitHub Actions Security Knowledge Base

If you are the owner of a GitHub Action, please contribute information about the use of `GITHUB_TOKEN` for your Action. 

# How do I contribute to the knowledge base?

To contribute information about the use of `GITHUB_TOKEN` for your Action:

1. Add a folder for your GitHub Action under the [`knowledge-base/actions`](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/) folder. It should match the path of your GitHub Action's `action.yml` file. As an example, 
   - If your GitHub Action's `action.yml` file is at the root, e.g. https://github.com/stelligent/cfn_nag/blob/master/action.yml, the path should be `knowledge-base/actions/stelligent/cfn_nag`
   - If your GitHub Action's `action.yml` file is in a sub folder, e.g. at https://github.com/snyk/actions/blob/master/gradle/action.yml, the path should be `knowledge-base/actions/snyk/actions/gradle`
2. In the folder for your GitHub Action, add an `action-security.yml` file.
3. Add metadata in the `action-security.yml` file about the use of `GITHUB_TOKEN`.

## Basic Scenarios

### 1. Your Action does not use the GitHub token
For this scenario, 
1. Add a `name` attribute in your `action-security.yml` file. You can set the name to be same as the name in your `action.yml` file. 
2. In a comment just mention that the GitHub token is not used. 

Here is an [example](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/stelligent/cfn_nag/action-security.yml). 

``` yaml
name: 'Stelligent cfn_nag' # stelligent/cfn_nag
# GITHUB_TOKEN not used
```

Note: if your Action just uses `metadata` permission to overcome throttle limits, it falls into this scenario

### 2. Your Action uses the GitHub token

For this scenario, follow these steps:
1. Add a `name` attribute in your `action-security.yml` file. You can set the name to be same as the name in your `action.yml` file. 
2. Mention where you expect the GitHub token.
    - If you expect it as an environment variable, you specify it this way. Here is an [example](https://github.com/step-security/secure-repo/blob/00c05310c1c97a91b98c46f904e857a617a2fc02/knowledge-base/actions/dev-drprasad/delete-tag-and-release/action-security.yml):
    ``` yaml
    name: Delete tag and release
    github-token:
      environment-variable-name: GITHUB_TOKEN      
    ```
    - If you expect it as an action input, you specify it as shown below. If you set the default value for the token to be the GITHUB_TOKEN, then set the “is-default” attribute to true. Here is an [example](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/irongut/editrelease/action-security.yml):
    ``` yaml
    name: 'Edit Release'
    github-token:
      action-input:
        input: token
        is-default: false      
    ```
  3. Mention the permissions needed and a reason for the permissions. The reason must start with the word `to`. 
  Here is an [example](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/peter-evans/create-or-update-comment/action-security.yml):
  ``` yaml
  name: 'Create or Update Comment'
  github-token:
    action-input:
      input: token
      is-default: true
    permissions:
      issues: write
      issues-reason: to create or update comment
      pull-requests: write
      pull-requests-reason: to create or update comment
  ```

## Syntax for action-security.yml

The above two scenarios should take care of most of the cases. For more advanced cases, here is the detailed syntax for the action-security.yml file

## `name`

**Required** The name of your action, should be the same as in your GitHub Action's action.yml file

## `github-token`

**Optional** github-token allows you to specify where `GITHUB_TOKEN` is expected as input for your GitHub Action, what permissions it needs, and the reason for those permissions. If your Action does not use the `GITHUB_TOKEN`, you do not need to set this.

## Example

This example is for `peter-evans/close-issue` GitHub Action. It shows that the Action expects GitHub token as an action input, the name of the input is `token`, and that it is set to `GITHUB_TOKEN` as the default value. It also shows that the permissions needed for the Action are `issues: write` and the reason for that permission is specified in the `issues-reason` key.

[`knowledge-base/actions/peter-evans/close-issue/action-security.yml`](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/peter-evans/close-issue/action-security.yml)

``` yaml
github-token:
  action-input:
    input: token
    is-default: true
  permissions:
    issues: write
    issues-reason: to close issues
```

## `github-token.action-input`

**Optional** github-token.action-input allows you to specify which input is used to accept the `GITHUB_TOKEN`. If your Action expects the token to be specified as an input to your Action, specify it in the `input` key. If you accept the `GITHUB_TOKEN` in environment variable, you do not need to set this.

## `github-token.action-input.input`

**Required** github-token.action-input.input should be the name of the input in your `action.yml` file that accepts the `GITHUB_TOKEN`. This is required if `action-input` is set.

## `github-token.action-input.is-default`

**Required** github-token.action-input.is-default This is required if `action-input` is set. It should be set to true, if `default` value of the input that expects the `GITHUB_TOKEN` is `${{ github.token }}`.

## `github-token.environment-variable-name`

**Optional** If you expect the `GITHUB_TOKEN` to be set in an environment variable, specify the name of the environment variable in `environment-variable-name`. If you accept the `GITHUB_TOKEN` as an Action input, you do not need to set this.

## Example

This example is for `github/super-linter` GitHub Action. It shows that the Action expects GitHub token as an environment variable, the name of the environment variable is `GITHUB_TOKEN`. It also shows that the permissions needed for the Action are `statuses: write` and the reason for that permission is specified in the `statuses-reason` key.

[`knowledge-base/actions/github/super-linter/action-security.yml`](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/github/super-linter/action-security.yml)

``` yaml
name: 'Super-Linter'
github-token:
  environment-variable-name: GITHUB_TOKEN
  permissions:
    statuses: write
    statuses-reason: to mark status of each linter run
```

## `github-token.permissions`

**Optional** If your Action uses the `GITHUB_TOKEN` provide information about the permissions needed using `github-token.permissions`. Each permission that is needed should be specified as a scope. If you only use the token to prevent rate-limiting, it needs `metadata` permissions, which the token has by default, so you do not need to provide additional scopes.

## Example

This example is for `actions/setup-node` GitHub Action. It shows that the Action expects GitHub token as an Action input. The permissions key is set, but no scopes are defined, since it only uses it for rate-limiting.

[`knowledge-base/actions/actions/setup-node/action-security.yml`](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/actions/setup-node/action-security.yml)

``` yaml
name: 'Setup Node.js environment'
github-token:
  action-input:
    input: token
    is-default: true
  permissions: # Used to pull node distributions from node-versions. Metadata permissions is present by default
```

## `github-token.permissions.<scope>`

**Optional** If your Action uses the `GITHUB_TOKEN` and uses it for a scope other than `metadata`, provide what scope is needed. Valid scopes are documented [here](https://docs.github.com/en/actions/security-guides/automatic-token-authentication#permissions-for-the-github_token).

## `github-token.permissions.<scope>-reason`

**Optional** If your Action uses the `GITHUB_TOKEN` and uses it for a scope other than `metadata`, provide the reason you need the scope. This information gets added to a workflow when its permissions are calculated automatically using the knowledge base. This is required if the corresponding scope is set in the permissions. The reason must start with `"to "`.

## Example

As an example, consider this `action-security.yml` for `peter-evans/close-issue` GitHub Action.

[`knowledge-base/actions/peter-evans/close-issue/action-security.yml`](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/peter-evans/close-issue/action-security.yml)

``` yaml
github-token:
  action-input:
    input: token
    is-default: true
  permissions:
    issues: write
    issues-reason: to close issues
```

When a GitHub workflow uses `peter-evans/close-issue` GitHub Action, and it uses the knowledge base to set permissions automatically, this is the output. The `permissions` key is added to the job, and the reason is added in a comment. The reason is taken from the `github-token.permissions.<scope>-reason` value from the knowledge base.

``` yaml
jobs:
  closeissue:
    permissions:
      issues: write # for peter-evans/close-issue@v1 to close issues
    runs-on: ubuntu-latest

    steps:
    - name: Close Issue
      uses: peter-evans/close-issue@v1
      with:
       issue-number: 1
       comment: Auto-closing issue
```

## `github-token.permissions.<scope>-if`

**Optional** If your Action uses the `GITHUB_TOKEN` but certain scopes are used only under certain conditions, the condition can be specified using `<scope>-if`.

## Example

As an example, consider this `action-security.yml` for `dessant/lock-threads` GitHub Action. The `issues` scope only applies if either the `with` (action input) does not have `process-only` or `process-only` is set to `issues`.

[`knowledge-base/actions/dessant/lock-threads/action-security.yml`](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/dessant/lock-threads/action-security.yml)

``` yaml
github-token:
  action-input:
    input: github-token
    is-default: true
  permissions:
    issues: write
    issues-if: ${{ !contains(with, 'process-only') || with['process-only'] == 'issues' }}
    issues-reason: to lock issues
    pull-requests: write
    pull-requests-if: ${{ !contains(with, 'process-only') || with['process-only'] == 'prs' }}
    pull-requests-reason: to lock PRs
```
