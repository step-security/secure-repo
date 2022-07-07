package main

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const (
	HardenRunnerActionPathWithTag = "step-security/harden-runner@v1"
	HardenRunnerActionPath        = "step-security/harden-runner"
	HardenRunnerActionName        = "Harden Runner"
)

func SecureWorkflow(queryStringParams map[string]string, inputYaml string, svc dynamodbiface.DynamoDBAPI) (*SecureWorkflowReponse, error) {
	pinActions, addHardenRunner, addPermissions := true, true, true
	pinnedActions, addedHardenRunner, addedPermissions := false, false, false

	if queryStringParams["pinActions"] == "false" {
		pinActions = false
	}

	if queryStringParams["addHardenRunner"] == "false" {
		addHardenRunner = false
	}

	if queryStringParams["addPermissions"] == "false" {
		addPermissions = false
	}

	secureWorkflowReponse := &SecureWorkflowReponse{FinalOutput: inputYaml, OriginalInput: inputYaml}
	var err error
	if addPermissions {
		secureWorkflowReponse, err = AddJobLevelPermissions(secureWorkflowReponse.FinalOutput)
		secureWorkflowReponse.OriginalInput = inputYaml
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

		addedPermissions = !secureWorkflowReponse.AlreadyHasPermissions
	}

	if addHardenRunner {
		secureWorkflowReponse.FinalOutput, addedHardenRunner, _ = AddAction(secureWorkflowReponse.FinalOutput, HardenRunnerActionPathWithTag)
	}

	if pinActions {
		pinnedAction, pinnedDocker := false, false
		secureWorkflowReponse.FinalOutput, pinnedAction, _ = PinActions(secureWorkflowReponse.FinalOutput)
		secureWorkflowReponse.FinalOutput, pinnedDocker, _ = PinDocker(secureWorkflowReponse.FinalOutput)
		pinnedActions = pinnedAction || pinnedDocker
	}

	// Setting appropriate flags
	secureWorkflowReponse.PinnedActions = pinnedActions
	secureWorkflowReponse.AddedHardenRunner = addedHardenRunner
	secureWorkflowReponse.AddedPermissions = addedPermissions
	return secureWorkflowReponse, nil
}
