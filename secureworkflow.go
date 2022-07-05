package main

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const (
	HardenRunnerActionPathWithTag = "step-security/harden-runner@v1"
	HardenRunnerActionPath        = "step-security/harden-runner"
	HardenRunnerActionName        = "Harden Runner"
	NeedsFixValue                 = "Yes"
	DoesNotNeedFixValue           = "No"
	NotApplicableValue            = "N/A"
)

func convertBoolToFixValue(val bool) string {
	if val {
		return NeedsFixValue
	}
	return DoesNotNeedFixValue
}

func SecureWorkflow(queryStringParams map[string]string, inputYaml string, svc dynamodbiface.DynamoDBAPI) (*SecureWorkflowReponse, error) {
	pinActions := true
	addHardenRunner := true
	addPermissions := true
	needsRestrictedToken := NotApplicableValue
	needsDepdendencyPinning := NotApplicableValue
	needsHardenRunnerAction := NotApplicableValue

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

		if secureWorkflowReponse.AlreadyHasPermissions {
			needsRestrictedToken = DoesNotNeedFixValue
		} else {
			needsRestrictedToken = NeedsFixValue
		}
	}

	if addHardenRunner {
		updated := false
		secureWorkflowReponse.FinalOutput, updated, _ = AddAction(secureWorkflowReponse.FinalOutput, HardenRunnerActionPathWithTag)
		needsHardenRunnerAction = convertBoolToFixValue(updated)
	}

	if pinActions {
		pinnedAction, pinnedDocker := false, false
		secureWorkflowReponse.FinalOutput, pinnedAction, _ = PinActions(secureWorkflowReponse.FinalOutput)
		secureWorkflowReponse.FinalOutput, pinnedDocker, _ = PinDocker(secureWorkflowReponse.FinalOutput)
		needsDepdendencyPinning = convertBoolToFixValue(pinnedAction || pinnedDocker)
	}

	// Setting appropriate flags
	secureWorkflowReponse.NeedsDepdendencyPinning = needsDepdendencyPinning
	secureWorkflowReponse.NeedsHardenRunnerAction = needsHardenRunnerAction
	secureWorkflowReponse.NeedsRestrictedToken = needsRestrictedToken
	return secureWorkflowReponse, nil

}
