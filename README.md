[![secure-workflows](images/banner.png)](#)
[![codecov](https://codecov.io/gh/step-security/secure-workflows/branch/main/graph/badge.svg?token=02ONA6U92A)](https://codecov.io/gh/step-security/secure-workflows)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://raw.githubusercontent.com/step-security/secure-workflows/main/LICENSE)

---

An open platform to update your CI/CD pipelines to comply with security requirements.

If you use GitHub Actions, use can use SecureWorkflows to:

- [Automatically set minimum GITHUB_TOKEN permissions](#1-automatically-set-minimum-github_token-permissions)
- [Pin Actions to a full length commit SHA](2-pin-actions-to-a-full-length-commit-sha)
- [Add Harden-Runner GitHub Action to each job](#3-add-harden-runner-github-action-to-each-job)

Support for GitLab, CircleCI, and more CI/CD providers will be added in the future. Check the [Roadmap](#roadmap) for details.

## In the News

- SecureWorkflows was used to secure critical open source projects
- StepSecurity was rewarded a [Secure Open Source (SOS) reward](https://sos.dev) for this work
- Secure-Workflows to be demoed at SupplyChainSecurityCon, Open Source Summit ([Link to Presentation](http://sched.co/11Pvu))

## Quickstart

### Using app.stepsecurity.io

To secure your GitHub Actions workflow:

- Copy and paste your GitHub Actions workflow YAML file at https://app.stepsecurity.io
- Click `Secure Workflows` button
- Paste the fixed workflow back in your codebase

GitHub App to create pull requests will be released soon. Check the [Roadmap](#roadmap) for details.

<p align="left">
  <img src="https://github.com/step-security/supply-chain-goat/blob/main/images/secure-workflows/SecureWorkflows4.gif" alt="Secure workflow screenshot" >
</p>

### Integration with OpenSSF Scorecard

- Add [OpenSSF Scorecards](https://github.com/ossf/scorecard-action) starter workflow
- View the Scorecard results in GitHub Code Scanning UI
- Follow remediation tip that points to https://app.stepsecurity.io

<p align="left">
  <img src="https://github.com/step-security/supply-chain-goat/blob/main/images/secure-workflows/SecureWorkflowsIntegration.png" alt="Secure workflow Scorecard integration screenshot" >
</p>

## Functionality Overview

Secure-Workflows API

- Takes in a GitHub Actions workflow YAML file as an input
- Returns a transformed workflow file with fixes applied
- You can select which of these changes you want to make

### 1. Automatically set minimum GITHUB_TOKEN permissions

#### Why is this needed?

- The GITHUB_TOKEN is an automatically generated secret to make authenticated calls to the GitHub API
- If the token is compromised, it can be misused (e.g. to overwrite releases or source code)
- To limit the damage, [GitHub recommends setting minimum token permissions for the GITHUB_TOKEN](https://github.blog/changelog/2021-04-20-github-actions-control-permissions-for-github_token/).

#### Before and After the fix

Before the fix, your workflow may look like this (no permissions set)

```yaml
on:
  push:

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
on:
  push:

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

#### How does Secure-Workflows fix this issue?

- Secure-Workflows stores the permissions needed by different GitHub Actions in a [knowledge base](<(https://github.com/step-security/secure-workflows/tree/main/knowledge-base/actions)>)
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
on:
  pull_request:

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
on:
  pull_request:

jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@544eadc6bf3d226fd7a7a9f0dc5b5bf7ca0675b9
      - name: Integration test
        uses: docker://ghcr.io/step-security/integration-test/int@sha256:1efef3bbdd297d1b321b9b4559092d3131961913bc68b7c92b681b4783d563f0
```

#### How does Secure-Workflows fix this issue?

- Secure-Workflows automates the process of getting the commit SHA for each mutable Action version or Docker image tag
- It does this by using GitHub and Docker registry APIs

### 3. Add Harden-Runner GitHub Action to each job

#### Why is this needed?

[Harden-Runner GitHub Action](https://github.com/step-security/harden-runner) installs a security agent on the Github-hosted runner to prevent exfiltration of credentials, monitor the build process, and detect compromised dependencies.

#### Before and After the fix

Before the fix, your workflow may look like this

```yaml
on:
  push:

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
on:
  push:

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

#### How does Secure-Workflows fix this issue?

Secure-Workflows updates the YAML file and adds [Harden-Runner GitHub Action](https://github.com/step-security/harden-runner) as the first step to each job.

## Roadmap

- GitHub App to create pull requests to fix issues
- Support for GitLab CI YAML files
- Support for CircleCI YAML files
