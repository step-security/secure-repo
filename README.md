# SecureWorkflows 

[![codecov](https://codecov.io/gh/step-security/secure-workflows/branch/main/graph/badge.svg?token=02ONA6U92A)](https://codecov.io/gh/step-security/secure-workflows)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://raw.githubusercontent.com/step-security/secure-workflows/main/LICENSE)

Secure Workflows is an open-source API to secure GitHub Actions workflows by automatically updating the workflow (YAML) files.

The API takes in a GitHub Actions workflow file as an input and returns a transformed workflow YAML file with the following changes:
1. Minimum `GITHUB_TOKEN` permissions are set for each job
2. Actions are pinned to a full length commit SHA
3. Step Security [Harden Runner](https://github.com/step-security/harden-runner) GitHub Action is added to each job

[GitHub Actions Hardening Guide](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions) recommends #1 and #2 as security best practices. [OSSF Scorecards](https://opensource.googleblog.com/2020/11/security-scorecards-for-open-source.html) recommends using [SecureWorkflows](https://app.stepsecurity.io/) for [#1](https://github.com/ossf/scorecard/blob/main/docs/checks.md#token-permissions) and [#2](https://github.com/ossf/scorecard/blob/main/docs/checks.md#pinned-dependencies). 

Harden-Runner GitHub Action (#3) installs a security agent on the Github-hosted runner to prevent exfiltration of credentials, monitor the build process, and detect compromised dependencies.

## GitHub Actions Security Knowledge Base

To calculate minimum token permissions for a given workflow, a [Knowledge Base of GitHub Actions](https://github.com/step-security/secure-workflows/tree/main/knowledge-base) has been setup. The knowledge base has information about what permissions a GitHub Action needs when using the `GITHUB_TOKEN`. 

If you are the owner of a GitHub Action, please [contribute to the knowledge base](https://github.com/step-security/secure-workflows/blob/main/knowledge-base/README.md). This will increase trust for your GitHub Action and more developers would be comfortable using it, and it will improve security for everyone's GitHub Actions workflows.

## Try SecureWorkflows

To use SecureWorkflows, visit https://app.stepsecurity.io/

<p align="left">
  <img src="https://step-security-images.s3.us-west-2.amazonaws.com/secureworkflownew.png" alt="Secure workflow screenshot" >
</p>

[Twitter handle]: https://img.shields.io/twitter/follow/step_security.svg?style=social&label=Follow
[Twitter badge]: https://twitter.com/intent/follow?screen_name=step_security
