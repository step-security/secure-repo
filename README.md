<p align="center"><img src="images/banner1.png" height="80" /></p>

<p align="center">
Secure your GitHub repo with ease through automated security fixes
</p>

<div align="center">

[![Maintained by stepsecurity.io](https://img.shields.io/badge/maintained%20by-stepsecurity.io-blueviolet)](https://stepsecurity.io/?utm_source=github&utm_medium=organic_oss&utm_campaign=secure-repo)
[![Go Report Card](https://goreportcard.com/badge/github.com/step-security/secure-repo)](https://goreportcard.com/report/github.com/step-security/secure-repo)
[![codecov](https://codecov.io/gh/step-security/secure-repo/branch/main/graph/badge.svg?token=02ONA6U92A)](https://codecov.io/gh/step-security/secure-repo)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/step-security/secure-repo/badge)](https://api.securityscorecards.dev/projects/github.com/step-security/secure-repo)

</div>

<p align="center">
  <img src="images/secure-repo.gif" alt="Secure repo screenshot" >
</p>

<h3>
  <a href="#quickstart">Quickstart</a>
  <span> • </span>
  <a href="#functionality-overview">Functionality</a> 
   <span> • </span>
  <a href="#contributing">Contributing</a>  
</h3>

## Quickstart

### Hosted Instance: [app.stepsecurity.io/securerepo](https://app.stepsecurity.io/securerepo)

To secure your GitHub repo using a pull request:

- Go to https://app.stepsecurity.io/securerepo and enter your public GitHub repository
- Log in using your GitHub Account (no need to install any App or grant `write` access)
- View recommendations and click `Create pull request.` Here is an example pull request: https://github.com/electron/electron/pull/36343.

### Integration with OpenSSF Scorecard

- Add [OpenSSF Scorecards](https://github.com/ossf/scorecard-action) starter workflow
- View the Scorecard results in GitHub Code Scanning UI
- Follow the remediation tip that points to https://app.stepsecurity.io

<p align="center">
  <img src="images/SecureWorkflowsIntegration.png" alt="Secure repo Scorecard integration screenshot" width="600">
</p>

### Self Hosted

To create an instance of Secure Workflows, deploy _cloudformation/ecr.yml_ and _cloudformation/resources.yml_ CloudFormation templates in your AWS account. You can take a look at _.github/workflows/release.yml_ for reference.

## Functionality

1. [Automatically set minimum GITHUB_TOKEN permissions](#1-automatically-set-minimum-github_token-permissions)
2. [Add Harden-Runner GitHub Action to each job](#3-add-harden-runner-github-action-to-each-job)
3. [Pin Actions to a full length commit SHA](#2-pin-actions-to-a-full-length-commit-sha)
4. [Pin image tags to digests in Dockerfiles]()
5. [Add or update Dependabot configuration](#4-add-or-update-dependabot-configuration)
6. [Add CodeQL workflow (SAST)](#5-add-codeql-workflow-sast)

### 1. Automatically set minimum GITHUB_TOKEN permissions

#### Why is this needed?

- The GITHUB_TOKEN is an automatically generated secret to make authenticated calls to the GitHub API
- If the token is compromised, it can be abused to compromise your environment (e.g., to overwrite releases or source code). This compromise will also impact everyone using your software in their supply chain.
- To limit the damage, [GitHub recommends setting minimum token permissions for the GITHUB_TOKEN](https://github.blog/changelog/2021-04-20-github-actions-control-permissions-for-github_token/).

#### Before and After the fix

**Pull request example**: https://github.com/nginxinc/kubernetes-ingress/pull/3134

In this pull request, minimum permissions are set automatically for the GITHUB_TOKEN

<p align="center"><img src="images/token-perm-example.png" alt="Screenshot of token permissions set in a workflow" width="600" /></p>

#### How does Secure-Repo fix this issue?

- Secure-Repo stores the permissions needed by different GitHub Actions in a [knowledge base](<(https://github.com/step-security/secure-repo/tree/main/knowledge-base/actions)>)
- It looks up the permissions needed by each Action in your workflow and sums the permissions up to come up with a final recommendation
- If you are the owner of a GitHub Action, please [contribute to the knowledge base](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/README.md)

### 2. Add Harden-Runner GitHub Action to each job

#### Why is this needed?

[Harden-Runner GitHub Action](https://github.com/step-security/harden-runner) installs a security agent on the Github-hosted runner to prevent exfiltration of credentials, monitor the build process, and detect compromised dependencies.

#### Before and After the fix

**Pull request example**: https://github.com/python-attrs/attrs/pull/1034

This pull request adds the Harden Runner GitHub Action to the workflow file.

<p align="center"><img src="images/harden-runner-example.png" width="600" alt="Screenshot of Harden-Runner GitHub Action added to a workflow" /></p>

#### How does Secure-Repo fix this issue?

Secure-Repo updates the YAML file and adds [Harden-Runner GitHub Action](https://github.com/step-security/harden-runner) as the first step to each job.

### 3. Pin Actions to a full length commit SHA

#### Why is this needed?

- GitHub Action tags and Docker tags are mutable, which poses a security risk
- If the tag changes you will not have a chance to review the change before it gets used
- GitHub's Security Hardening for GitHub Actions guide [recommends pinning actions to full length commit for third party actions](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#using-third-party-actions).

#### Before and After the fix

Before the fix, your workflow may look like this (use of `v1` and `latest` tags)

After the fix, Secure-Repo pins each Action and docker image to an immutable checksum.

**Pull request example**: https://github.com/electron/electron/pull/36343

In this pull request, the workflow file has the GitHub Actions tags pinned automatically to their full-length commit SHA.

<p align="center"><img src="images/pin-example.png" alt="Screenshot of Action pinned to commit SHA" width="600" /></p>

#### How does Secure-Repo fix this issue?

- Secure-Repo automates the process of getting the commit SHA for each mutable Action version or Docker image tag
- It does this by using GitHub and Docker registry APIs

### 4. Pin image tags to digests in Dockerfiles

#### Why is this needed?

- Docker tags are mutable, so use digests in place of tags when pulling images
- If the tag changes you will not have a chance to review the change before it gets used
- OpenSSF Scorecard [recommends pinning image tags for Dockerfiles used in building and releasing your project](https://github.com/ossf/scorecard/blob/main/docs/checks.md#pinned-dependencies).

#### Before and After the fix

Before the fix, your Dockerfile uses image:tag, e.g. `rust:latest`

After the fix, Secure-Repo pins each docker image to an immutable checksum, e.g. `rust:latest@sha256:02a53e734724bef4a58d856c694f826aa9e7ea84353516b76d9a6d241e9da60e`.

**Pull request example**: https://github.com/fleetdm/fleet/pull/10205

In this pull request, the Docker file has tags pinned automatically to their checksum.

<p align="center"><img src="images/pin-docker-example.png" alt="Screenshot of docker image pinned to checksum" width="600" /></p>

#### How does Secure-Repo fix this issue?

- Secure-Repo automates the process of getting the checksum for each Docker image tag
- It does this by using Docker registry APIs

### 5. Add or update Dependabot configuration

#### Why is this needed?

- You enable Dependabot version updates by checking a `dependabot.yml` configuration file into your repository
- Dependabot ensures that your repository automatically keeps up with the latest releases of the packages and applications it depends on

#### Before and After the fix

Before the fix, you might not have a `dependabot.yml` file or it might not cover all ecosystems used in your project.

After the fix, the `dependabot.yml` file is added or updated with configuration for all package ecosystems used in your project.

**Pull request example**: https://github.com/muir/libschema/pull/31

This pull request updates the Dependabot configuration.

<p align="center"><img src="images/dependabot-example.png" width="600" alt="Screenshot of Dependabot config updated" /></p>

#### How does Secure-Repo fix this issue?

Secure-Repo updates the `dependabot.yml` file to add missing ecosystems. For example, if the Dependabot configuration updates npm packages but not GitHub Actions, it is updated to add the GitHub Actions ecosystem.

### 6. Add CodeQL workflow (SAST)

#### Why is this needed?

- Using Static Application Security Testing (SAST) tools can prevent known classes of bugs from being introduced in the codebase

#### Before and After the fix

Before the fix, you do not have a CodeQL workflow.

After the fix, a `codeql.yml` GitHub Actions workflow gets added to your project.

**Pull request example**: https://github.com/rubygems/rubygems.org/pull/3314

This pull request adds CodeQL to the list of workflows.

#### How does Secure-Repo fix this issue?

Secure-Repo has a [workflow-templates](https://github.com/step-security/secure-repo/tree/main/workflow-templates) folder. This folder has the default CodeQL workflow, which gets added as part of the pull request. The placeholder for languages in the template gets replaced with languages for your GitHub repository.

## Contributing

Contributions are welcome!

If you are the owner of a GitHub Action, please contribute information about the use of GITHUB_TOKEN for your Action. This will enable the community to automatically calculate minimum token permissions for the GITHUB_TOKEN for their workflows. Check out the [Contributing Guide](https://github.com/step-security/secure-repo/blob/main/knowledge-base/actions/README.md)
