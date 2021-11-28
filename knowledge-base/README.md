# Contribute to the GitHub Actions security knowledge base

If you are the owner of a GitHub Action, please contribute information about the use of `GITHUB_TOKEN` and expected outbound calls for your Action. 

This will enable the community to:
1. Automatically calculate minimum token permissions for the `GITHUB_TOKEN` for their workflows. 
2. Restrict outbound traffic for their GitHub Actions worklows to allowed endpoints.

This will increase trust for your GitHub Action and more developers would be comfortable using it, and it will improve security for everyone's GitHub Actions workflows.

# How do I contribute to the knowledge base?

To contibute to the knowledge base:
1. Add a folder under the knowledge-base folder for your GitHub Action.
2. In the folder for your GitHub Action, add an action-security.yml file. You can view existing files to understand the structure of these YAML files. 
3. Add metadata in the action-security.yml file about the use of `GITHUB_TOKEN` and expected outbound traffic for your GitHub Action.