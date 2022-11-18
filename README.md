<p align="center"><img src="images/banner.png" height="80" /></p>

<h1 align="center">Secure Workflows</h1>

<p align="center">
Secure GitHub Actions CI/CD workflows via automated remediations
</p>

<div align="center">

[![Maintained by stepsecurity.io](https://img.shields.io/badge/maintained%20by-stepsecurity.io-blueviolet)](https://stepsecurity.io/?utm_source=github&utm_medium=organic_oss&utm_campaign=secure-workflows)
[![Go Report Card](https://goreportcard.com/badge/github.com/step-security/secure-workflows)](https://goreportcard.com/report/github.com/step-security/secure-workflows)
[![codecov](https://codecov.io/gh/step-security/secure-workflows/branch/main/graph/badge.svg?token=02ONA6U92A)](https://codecov.io/gh/step-security/secure-workflows)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/step-security/secure-workflows/badge)](https://api.securityscorecards.dev/projects/github.com/step-security/secure-workflows)

</div>

<p align="center">
  <img src="https://github.com/step-security/supply-chain-goat/blob/main/images/secure-repo.gif" alt="Secure repo screenshot" >
</p>

<h3>
  <a href="#quickstart">Quickstart</a>
  <span> • </span>
  <a href="#functionality-overview">Functionality Overview</a> 
   <span> • </span>
  <a href="#contributing">Contributing</a>  
</h3>

## Quickstart

### Hosted Instance: [app.stepsecurity.io/securerepo](https://app.stepsecurity.io/securerepo)

To secure GitHub Actions workflows using a pull request:

- Go to https://app.stepsecurity.io/securerepo and enter your public GitHub repository
- Login using your GitHub Account (no need to install any App or grant `write` access)
- View recommendations and click `Create pull request`. Here is a [sample pull request](https://github.com/Kapiche/cobertura-action/pull/60).

### Integration with OpenSSF Scorecard

- Add [OpenSSF Scorecards](https://github.com/ossf/scorecard-action) starter workflow
- View the Scorecard results in GitHub Code Scanning UI
- Follow remediation tip that points to https://app.stepsecurity.io

<p align="center">
  <img src="images/SecureWorkflowsIntegration.png" alt="Secure workflow Scorecard integration screenshot" width="600">
</p>

### Self Hosted

To create an instance of Secure Workflows, deploy _cloudformation/ecr.yml_ and _cloudformation/resources.yml_ CloudFormation templates in your AWS account. You can take a look at _.github/workflows/release.yml_ for reference.

## Functionality Overview

Secure Workflows

- Takes in a GitHub Actions workflow YAML file as an input
- Returns a transformed workflow file with fixes applied
- You can select which of these changes you want to make

### 1. Automatically set minimum GITHUB_TOKEN permissions

#### Why is this needed?

- The GITHUB_TOKEN is an automatically generated secret to make authenticated calls to the GitHub API
- If the token is compromised, it can be abused to compromise your environment (e.g. to overwrite releases or source code). This will also impact everyone who use your software in their software supply chain.
- To limit the damage, [GitHub recommends setting minimum token permissions for the GITHUB_TOKEN](https://github.blog/changelog/2021-04-20-github-actions-control-permissions-for-github_token/).

#### Before and After the fix

**Pull request example**: https://github.com/nginxinc/kubernetes-ingress/pull/3134

In this pull request, minimum permissions are set automatically for the GITHUB_TOKEN

<p align="center"><img src="images/token-perm-example.png" alt="Screenshot of token permissions set in a workflow" width="600" /></p>

#### How does SecureWorkflows fix this issue?

- SecureWorkflows stores the permissions needed by different GitHub Actions in a [knowledge base](<(https://github.com/step-security/secure-workflows/tree/main/knowledge-base/actions)>)
- It looks up the permissions needed by each Action in your workflow, and sums the permissions up to come up with a final recommendation
- If you are the owner of a GitHub Action, please [contribute to the knowledge base](https://github.com/step-security/secure-workflows/blob/main/knowledge-base/actions/README.md)

### 2. Pin Actions to a full length commit SHA

#### Why is this needed?

- GitHub Action tags and Docker tags are mutatble. This poses a security risk
- If the tag changes you will not have a chance to review the change before it gets used
- GitHub's Security Hardening for GitHub Actions guide [recommends pinning actions to full length commit for third party actions](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#using-third-party-actions).

#### Before and After the fix

Before the fix, your workflow may look like this (use of `v1` and `latest` tags)

After the fix, each Action and docker image will be pinned to an immutable checksum.

**Pull request example**: https://github.com/electron/electron/pull/36343

In this pull request, the workflow file has the GitHub Actions tags pinned automatically to their full-length commit SHA.

<p align="center"><img src="images/pin-example.png" alt="Screenshot of Action pinned to commit SHA" width="600" /></p>

#### How does SecureWorkflows fix this issue?

- SecureWorkflows automates the process of getting the commit SHA for each mutable Action version or Docker image tag
- It does this by using GitHub and Docker registry APIs

### 3. Add Harden-Runner GitHub Action to each job

#### Why is this needed?

[Harden-Runner GitHub Action](https://github.com/step-security/harden-runner) installs a security agent on the Github-hosted runner to prevent exfiltration of credentials, monitor the build process, and detect compromised dependencies.

#### Before and After the fix

**Pull request example**: https://github.com/python-attrs/attrs/pull/1034

This pull request adds the Harden Runner GitHub Action to the workflow file.

<p align="center"><img src="images/harden-runner-example.png" width="600" alt="Screenshot of Harden-Runner GitHub Action added to a workflow" /></p>

#### How does SecureWorkflows fix this issue?

SecureWorkflows updates the YAML file and adds [Harden-Runner GitHub Action](https://github.com/step-security/harden-runner) as the first step to each job.

## Contributing

Contributions are welcome!

If you are the owner of a GitHub Action, please contribute information about the use of GITHUB_TOKEN for your Action. This will enable the community to automatically calculate minimum token permissions for the GITHUB_TOKEN for their workflows. Check out the [Contributing Guide](https://github.com/step-security/secure-workflows/blob/main/knowledge-base/actions/README.md)
