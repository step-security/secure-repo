package main

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const (
	HardenRunnerActionPathWithTag = "step-security/harden-runner@v1"
	HardenRunnerActionPath        = "step-security/harden-runner"
	HardenRunnerActionName        = "Harden Runner"
)

func SecureWorkflow(queryStringParams map[string]string, inputYaml string, svc dynamodbiface.DynamoDBAPI) (*SecureWorkflowReponse, error, bool) {
	pinActions := true
	addHardenRunner := true
	addPermissions := true
	isResolved := false

	if queryStringParams["pinActions"] == "false" {
		pinActions = false
	}

	if queryStringParams["addHardenRunner"] == "false" {
		addHardenRunner = false
	}

	if queryStringParams["addPermissions"] == "false" {
		addPermissions = false
	}

	secureWorkflowReponse := &SecureWorkflowReponse{FinalOutput: inputYaml}
	var err error
	if addPermissions {
		secureWorkflowReponse, err = AddJobLevelPermissions(secureWorkflowReponse.FinalOutput)
		if err != nil {
			return nil, err, false
		} else {
			if !secureWorkflowReponse.HasErrors {
				secureWorkflowReponse.FinalOutput, _ = AddWorkflowLevelPermissions(secureWorkflowReponse.FinalOutput)
				
			}
			if len(secureWorkflowReponse.MissingActions) > 0 {
				StoreMissingActions(secureWorkflowReponse.MissingActions, svc)

			}
			
			isResolved = true
		}
	}

	if addHardenRunner {
		secureWorkflowReponse.FinalOutput, _ = AddAction(secureWorkflowReponse.FinalOutput, HardenRunnerActionPathWithTag)
	}

	if pinActions {
		secureWorkflowReponse.FinalOutput, _ = PinActions(secureWorkflowReponse.FinalOutput)
	}

	return secureWorkflowReponse, nil, isResolved

}
