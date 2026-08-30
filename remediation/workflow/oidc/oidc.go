package oidc

import (
	"fmt"
	"strings"

	metadata "github.com/step-security/secure-repo/remediation/workflow/metadata"
	"github.com/step-security/secure-repo/remediation/workflow/permissions"
	"gopkg.in/yaml.v3"
)

func UpdateActionsToUseOIDC(inputYaml string) (string, bool, error) {
	updatedYaml, updated, err := updateActionsToUseOIDC(inputYaml)
	if err != nil {
		return "", false, fmt.Errorf("failed to update actions to use OIDC: %w", err)
	}

	if !updated {
		return "", false, fmt.Errorf("inputYaml does not have OIDC actions")
	}

	secureWorkflowReponse, err := permissions.AddJobLevelPermissions(updatedYaml)
	if err != nil {
		return "", false, err
	}

	if !secureWorkflowReponse.HasErrors {
		return secureWorkflowReponse.FinalOutput, true, nil
	}

	workflow := metadata.Workflow{}
	err = yaml.Unmarshal([]byte(updatedYaml), &workflow)
	if err != nil {
		return "", false, err
	}

	IsContentRead := false
	if len(workflow.Permissions.Scopes) == 1 {
		if val, ok := workflow.Permissions.Scopes["contents"]; ok {
			if val == "read" {
				IsContentRead = true
			}
		}
	}

	for jobName, job := range workflow.Jobs {
		if val, ok := job.Permissions.Scopes["id-token"]; ok {
			if val == "write" {
				continue
			}
		}
		containsOIDCAction := false
		for _, step := range job.Steps {
			if strings.Contains(step.Uses, "aws-actions/configure-aws-credentials") {
				containsOIDCAction = true
				break
			}
		}
		if containsOIDCAction && (job.Permissions.IsSet || IsContentRead) {
			updatedYaml, err = addOIDCPermission(updatedYaml, jobName, job.Permissions.IsSet)
			if err != nil {
				return "", false, err
			}
		} else if containsOIDCAction && (!job.Permissions.IsSet && !IsContentRead) {
			return "", false, err
		}
	}
	return updatedYaml, true, nil
}

func addOIDCPermission(inputYaml, jobName string, isSet bool) (string, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", fmt.Errorf("unable to parse yaml %v", err)
	}

	jobNode := permissions.IterateNode(&t, jobName, "!!map", 0)
	if isSet {
		jobNode = permissions.IterateNode(&t, "permissions", "!!map", jobNode.Line)
	}

	if jobNode == nil {
		return "", fmt.Errorf("jobName %s not found in the input yaml", jobName)
	}

	inputLines := strings.Split(inputYaml, "\n")
	var output []string
	for i := 0; i < jobNode.Line-1; i++ {
		output = append(output, inputLines[i])
	}

	spaces := ""
	for i := 0; i < jobNode.Column-1; i++ {
		spaces += " "
	}

	if !isSet {
		output = append(output, spaces+fmt.Sprintf("permissions:"))
		spaces += " "
	}
	output = append(output, spaces+fmt.Sprintf("id-token: write"))

	for i := jobNode.Line - 1; i < len(inputLines); i++ {
		output = append(output, inputLines[i])
	}

	return strings.Join(output, "\n"), nil

}

func updateActionsToUseOIDC(inputYaml string) (string, bool, error) {
	workflow := metadata.Workflow{}
	finalOutput := inputYaml
	updated := false

	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return "", false, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	// Iterate over each job
	for jobName, job := range workflow.Jobs {
		// Iterate over each step in the job
		for _, step := range job.Steps {
			if strings.Contains(step.Uses, "aws-actions/configure-aws-credentials") {
				finalOutput, err = updateAWSCredentialsAction(jobName, step, finalOutput)
				if err != nil {
					return "", false, fmt.Errorf("failed to update AWS credentials action: %w", err)
				}
				updated = true
			}
		}
	}

	return finalOutput, updated, nil
}

func updateAWSCredentialsAction(jobName string, step metadata.Step, inputYaml string) (string, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", fmt.Errorf("unable to parse yaml %v", err)
	}

	jobNode := permissions.IterateNode(&t, jobName, "!!map", 0)
	jobNode = permissions.IterateNode(&t, "steps", "!!seq", jobNode.Line)

	jobNode = permissions.IterateNode(&t, "aws-access-key-id", "!!str", jobNode.Line)
	if jobNode == nil {
		return "", fmt.Errorf("jobName %s not found in the input yaml", jobName)
	}

	inputLines := strings.Split(inputYaml, "\n")
	var output []string
	for i := 0; i < jobNode.Line-1; i++ {
		output = append(output, inputLines[i])
	}
	for i := jobNode.Line; i < len(inputLines); i++ {
		output = append(output, inputLines[i])
	}

	inputYaml = strings.Join(output, "\n")

	inputYaml = strings.ReplaceAll(inputYaml, "aws-secret-access-key", "role-to-assume")
	inputYaml = strings.ReplaceAll(inputYaml, step.With["aws-secret-access-key"], "arn:aws:iam::{OICD_ID}:role/my-github-actions-role")

	return inputYaml, nil
}
