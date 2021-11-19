<p align="left">
  <img src="https://step-security-images.s3.us-west-2.amazonaws.com/Final-Logo-06.png" alt="Step Security Logo" width="340">
</p>

# Secure GitHub Actions Workflow files [![codecov](https://codecov.io/gh/step-security/secure-workflows/branch/main/graph/badge.svg?token=02ONA6U92A)](https://codecov.io/gh/step-security/secure-workflows)

This is an API to secure GitHub Actions Workflow files. You can use it by visiting https://app.stepsecurity.io/secureworkflow

The API takes in a GitHub Actions workflow file as an input and returns a transformed workflow file with the following changes:
1. Minimum `GITHUB_TOKEN` permissions are set for each job
2. Step Security [Harden Runner](https://github.com/step-security/harden-runner) GitHub Action is added to each job

