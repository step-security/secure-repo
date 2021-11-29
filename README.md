<p align="left">
  <img src="https://step-security-images.s3.us-west-2.amazonaws.com/Final-Logo-06.png" alt="Step Security Logo" width="340">
</p>

# Secure workflows using GitHub Actions Security Knowledge Base 

[![codecov](https://codecov.io/gh/step-security/secure-workflows/branch/main/graph/badge.svg?token=02ONA6U92A)](https://codecov.io/gh/step-security/secure-workflows)
[![Twitter handle][]][Twitter badge]

Secure Workflows is an API to secure GitHub Actions Workflow files. 

The API takes in a GitHub Actions workflow file as an input and returns a transformed workflow file with the following changes:
1. Minimum `GITHUB_TOKEN` permissions are set for each job
2. Step Security [Harden Runner](https://github.com/step-security/harden-runner) GitHub Action is added to each job
3. Actions are pinned to a full length commit SHA

To calculate minimum token permissions for a given workflow, and to set allowed endpoints for workflows, a Knowledge Base of GitHub Actions is maintained. The knowledge base has information about what permissions a GitHub Action needs when using the `GITHUB_TOKEN` and what outbound calls the GitHub Action is expected to make. 

The knowledge base enables you to:
1. Automatically calculate minimum token permissions for the `GITHUB_TOKEN` for your workflows. 
2. Restrict outbound traffic for your GitHub Actions workflows to allowed endpoints.

<p align="left">
  <img src="https://step-security-images.s3.us-west-2.amazonaws.com/secureworkflow.png" alt="Secure workflow screenshot" >
</p>

[Twitter handle]: https://img.shields.io/twitter/follow/step_security.svg?style=social&label=Follow
[Twitter badge]: https://twitter.com/intent/follow?screen_name=step_security