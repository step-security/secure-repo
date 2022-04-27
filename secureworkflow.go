package main

import (
	"strings"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const (
	HardenRunnerActionPathWithTag = "step-security/harden-runner@v1"
	HardenRunnerActionPath        = "step-security/harden-runner"
	HardenRunnerActionName        = "Harden Runner"
)

func SecureWorkflow(queryStringParams map[string]string, inputYaml string, svc dynamodbiface.DynamoDBAPI) (*SecureWorkflowReponse, error) {
	pinActions := true
	addHardenRunner := true
	addPermissions := true

	if queryStringParams["pinActions"] == "false" {
		pinActions = false
	}

	if queryStringParams["addHardenRunner"] == "false" {
		addHardenRunner = false
	}

	if queryStringParams["addPermissions"] == "false" {
		addPermissions = false
	}

	if !isLinuxRunner(inputYaml) {
		// this check will ensure that harden-runner will be added only
		// if action runs-on ubuntu machine.
		addHardenRunner = false
	}

	secureWorkflowReponse := &SecureWorkflowReponse{FinalOutput: inputYaml}
	var err error
	if addPermissions {
		secureWorkflowReponse, err = AddJobLevelPermissions(secureWorkflowReponse.FinalOutput)
		if err != nil {
			return nil, err
		} else {
			if !secureWorkflowReponse.HasErrors {
				secureWorkflowReponse.FinalOutput, _ = AddWorkflowLevelPermissions(secureWorkflowReponse.FinalOutput)
			}
			if len(secureWorkflowReponse.MissingActions) > 0 {
				StoreMissingActions(secureWorkflowReponse.MissingActions, svc)
			}
		}
	}

	if addHardenRunner {
		secureWorkflowReponse.FinalOutput, _ = AddAction(secureWorkflowReponse.FinalOutput, HardenRunnerActionPathWithTag)
	}

	if pinActions {
		secureWorkflowReponse.FinalOutput, _ = PinActions(secureWorkflowReponse.FinalOutput)
		secureWorkflowReponse.FinalOutput, _ = PinDocker(secureWorkflowReponse.FinalOutput)
	}

	return secureWorkflowReponse, nil

}

func isLinuxRunner(content string) bool {
	if strings.Contains(content, "strategy:") && strings.Contains(content, "matrix:") && strings.Contains(content, "os:") {
		// matrix runner
		i := strings.Index(content, "os:")
		return strings.Contains(getLine(content, i), "ubuntu")
	}
	// normal runner
	i := strings.Index(content, "runs-on:")
	return strings.Contains(getLine(content, i), "ubuntu")
}

func getLine(content string, start int) string {
	end := start
	for {
		if content[end] == '\n' {
			break
		}
		end += 1
	}
	return content[start:end]

}
