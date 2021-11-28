# Contribute to the GitHub Actions security knowledge base

If you are the owner of a GitHub Action, please contribute information about the use of `GITHUB_TOKEN` and expected outbound calls for your Action. 

This will enable the community to:
1. Automatically calculate minimum token permissions for the `GITHUB_TOKEN` for their workflows. 
2. Restrict outbound traffic for their GitHub Actions worklows to allowed endpoints.

This will increase trust for your GitHub Action and more developers would be comfortable using it, and it will improve security for everyone's GitHub Actions workflows.

# How do I contribute to the knowledge base?

To contibute to the knowledge base:
1. Add a folder under the knowledge-base folder for your GitHub Action.
2. In the folder for your GitHub Action, add an `action-security.yml` file. You can view existing files to understand the structure of these YAML files. 
3. Add metadata in the `action-security.yml` file about the use of `GITHUB_TOKEN` and expected outbound traffic for your GitHub Action.

## Syntax for action-security.yml

The metadata filename must be `action-security.yml`. It must be located in a folder for your GitHub Action under the `knowledge-base` folder, e.g. the location for `actions/checkout` is [`knowledge-base/actions/checkout/action-security.yml`](https://github.com/step-security/secure-workflows/blob/main/knowledge-base/actions/checkout/action-security.yml) The data in the metadata file defines the permissions needed for `GITHUB_TOKEN` for your Action and the outbound calls made by your Action.

## `name`

**Required** The name of your action, should be the same as in your GitHub Action's action.yml file

## `github-token`

**Optional** github-token allows you to specify where `GITHUB_TOKEN` is expected as input for your GitHub Action, what permissions it needs, and the reason for those permissions. If your Action does not use the `GITHUB_TOKEN`, you do not need to set this. 

## Example

This example is for `peter-evans/close-issue` GitHub Action. It shows that the Action expects GitHub token as an action input, the name of the input is `token`, and that it is set to `GITHUB_TOKEN` as the default value. It also shows that the permissions needed for the Action are `issues: write` and the reason for that permission is specified in the `issues-reason` key. 

```
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