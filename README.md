<p align="center">
<picture>
  <source media="(prefers-color-scheme: light)" srcset="images/banner.png" width="400">
  <img src="images/banner.png" width="400">
</picture>
</p>

<div align="center">

[![Maintained by stepsecurity.io](https://img.shields.io/badge/maintained%20by-stepsecurity.io-blueviolet)](https://stepsecurity.io/?utm_source=github&utm_medium=organic_oss&utm_campaign=secure-workflows)
[![codecov](https://codecov.io/gh/step-security/secure-workflows/branch/main/graph/badge.svg?token=02ONA6U92A)](https://codecov.io/gh/step-security/secure-workflows)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/step-security/secure-workflows/badge)](https://api.securityscorecards.dev/projects/github.com/step-security/secure-workflows)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://raw.githubusercontent.com/step-security/secure-workflows/main/LICENSE)

</div>

<p align="center">
Secure GitHub Actions CI/CD workflows via automated remediations
</p>

<p align="center">
  <img src="https://github.com/step-security/supply-chain-goat/blob/main/images/secure-repo.gif" alt="Secure repo screenshot" >
</p>

<h3>
  <a href="#quickstart">Quickstart</a>
  <span> • </span>
   <a href="#impact">Impact</a>
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

<p align="left">
  <img src="https://github.com/step-security/supply-chain-goat/blob/main/images/secure-workflows/SecureWorkflowsIntegration.png" alt="Secure workflow Scorecard integration screenshot" width="60%">
</p>

### Self Hosted

To create an instance of Secure Workflows, deploy _cloudformation/ecr.yml_ and _cloudformation/resources.yml_ CloudFormation templates in your AWS account. You can take a look at _.github/workflows/release.yml_ for reference.

## Impact

- SecureWorkflows has been used to [secure 30 of the top 100 critical open source projects](https://github.com/step-security/secure-workflows/issues/462)
- SecureWorkflows was demoed at `SupplyChainSecurityCon` at [Open Source Summit North America 2022](http://sched.co/11Pvu)

## Functionality Overview

SecureWorkflows API

- Takes in a GitHub Actions workflow YAML file as an input
- Returns a transformed workflow file with fixes applied
- You can select which of these changes you want to make

### 1. Automatically set minimum GITHUB_TOKEN permissions

#### Why is this needed?

- The GITHUB_TOKEN is an automatically generated secret to make authenticated calls to the GitHub API
- If the token is compromised, it can be abused to compromise your environment (e.g. to overwrite releases or source code). This will also impact everyone who use your software in their software supply chain.
- To limit the damage, [GitHub recommends setting minimum token permissions for the GITHUB_TOKEN](https://github.blog/changelog/2021-04-20-github-actions-control-permissions-for-github_token/).

#### Before and After the fix

Before the fix, your workflow may look like this (no permissions set)

```yaml
jobs:
  closeissue:
    runs-on: ubuntu-latest

    steps:
      - name: Close Issue
        uses: peter-evans/close-issue@v1
        with:
          issue-number: 1
          comment: Auto-closing issue
```

After the fix, the workflow will have minimum permissions added for the GITHUB token.

```yaml
permissions:
  contents: read

jobs:
  closeissue:
    permissions:
      issues: write # for peter-evans/close-issue to close issues
    runs-on: ubuntu-latest

    steps:
      - name: Close Issue
        uses: peter-evans/close-issue@v1
        with:
          issue-number: 1
          comment: Auto-closing issue
```

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

```yaml
jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v1
      - name: Integration test
        uses: docker://ghcr.io/step-security/integration-test/int:latest
```

After the fix, each Action and docker image will be pinned to an immutable checksum.

```yaml
jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@544eadc6bf3d226fd7a7a9f0dc5b5bf7ca0675b9
      - name: Integration test
        uses: docker://ghcr.io/step-security/integration-test/int@sha256:1efef3bbdd297d1b321b9b4559092d3131961913bc68b7c92b681b4783d563f0
```

#### How does SecureWorkflows fix this issue?

- SecureWorkflows automates the process of getting the commit SHA for each mutable Action version or Docker image tag
- It does this by using GitHub and Docker registry APIs

### 3. Add Harden-Runner GitHub Action to each job

#### Why is this needed?

[Harden-Runner GitHub Action](https://github.com/step-security/harden-runner) installs a security agent on the Github-hosted runner to prevent exfiltration of credentials, monitor the build process, and detect compromised dependencies.

#### Before and After the fix

Before the fix, your workflow may look like this

```yaml
jobs:
  closeissue:
    runs-on: ubuntu-latest

    steps:
      - name: Close Issue
        uses: peter-evans/close-issue@v1
        with:
          issue-number: 1
          comment: Auto-closing issue
```

After the fix, each workflow has the harden-runner Action added as the first step.

```yaml
jobs:
  closeissue:
    runs-on: ubuntu-latest

    steps:
      - name: Harden Runner
        uses: step-security/harden-runner@v1
        with:
          egress-policy: audit

      - name: Close Issue
        uses: peter-evans/close-issue@v1
        with:
          issue-number: 1
          comment: Auto-closing issue
```

#### How does SecureWorkflows fix this issue?

SecureWorkflows updates the YAML file and adds [Harden-Runner GitHub Action](https://github.com/step-security/harden-runner) as the first step to each job.

## Contributing

Contributions are welcome!

If you are the owner of a GitHub Action, please contribute information about the use of GITHUB_TOKEN for your Action. This will enable the community to automatically calculate minimum token permissions for the GITHUB_TOKEN for their workflows. Check out the [Contributing Guide](https://github.com/step-security/secure-workflows/blob/main/knowledge-base/actions/README.md)
