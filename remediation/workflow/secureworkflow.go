package workflow

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/step-security/secure-workflows/remediation/workflow/hardenrunner"
	"github.com/step-security/secure-workflows/remediation/workflow/permissions"
	"github.com/step-security/secure-workflows/remediation/workflow/pin"
)

const HardenRunnerActionPathWithTag = "step-security/harden-runner@v1"

func SecureWorkflow(queryStringParams map[string]string, inputYaml string, svc dynamodbiface.DynamoDBAPI) (*permissions.SecureWorkflowReponse, error) {
	pinActions, addHardenRunner, addPermissions, addProjectComment := true, true, true, true
	pinnedActions, addedHardenRunner, addedPermissions := false, false, false
	ignoreMissingKBs := false

	if queryStringParams["pinActions"] == "false" {
		pinActions = false
	}

	if queryStringParams["addHardenRunner"] == "false" {
		addHardenRunner = false
	}

	if queryStringParams["addPermissions"] == "false" {
		addPermissions = false
	}

	if queryStringParams["ignoreMissingKBs"] == "true" {
		ignoreMissingKBs = true
	}

	if queryStringParams["addProjectComment"] == "false" {
		addProjectComment = false
	}

	secureWorkflowReponse := &permissions.SecureWorkflowReponse{FinalOutput: inputYaml, OriginalInput: inputYaml}
	var err error
	if addPermissions {
		secureWorkflowReponse, err = permissions.AddJobLevelPermissions(secureWorkflowReponse.FinalOutput)
		secureWorkflowReponse.OriginalInput = inputYaml
		if err != nil {
			return nil, err
		} else {
			if !secureWorkflowReponse.HasErrors || permissions.ShouldAddWorkflowLevelPermissions(secureWorkflowReponse.JobErrors) {
				secureWorkflowReponse.FinalOutput, err = permissions.AddWorkflowLevelPermissions(secureWorkflowReponse.FinalOutput, addProjectComment)
				if err != nil {
					secureWorkflowReponse.HasErrors = true
				} else {
					// reset the error
					// this is done because workflow perms have been added
					// only job errors were that perms already existed
					secureWorkflowReponse.HasErrors = false
				}
			}
			if len(secureWorkflowReponse.MissingActions) > 0 && !ignoreMissingKBs {
				StoreMissingActions(secureWorkflowReponse.MissingActions, svc)
			}
		}
		// if there are no errors, then we must have added perms
		// if there are already perms at workflow level, that is treated as an error condition
		addedPermissions = !secureWorkflowReponse.HasErrors
	}

	if addHardenRunner {
		secureWorkflowReponse.FinalOutput, addedHardenRunner, _ = hardenrunner.AddAction(secureWorkflowReponse.FinalOutput, HardenRunnerActionPathWithTag)
	}

	if pinActions {
		pinnedAction, pinnedDocker := false, false
		secureWorkflowReponse.FinalOutput, pinnedAction, _ = pin.PinActions(secureWorkflowReponse.FinalOutput)
		secureWorkflowReponse.FinalOutput, pinnedDocker, _ = pin.PinDocker(secureWorkflowReponse.FinalOutput)
		pinnedActions = pinnedAction || pinnedDocker
	}

	// Setting appropriate flags
	secureWorkflowReponse.PinnedActions = pinnedActions
	secureWorkflowReponse.AddedHardenRunner = addedHardenRunner
	secureWorkflowReponse.AddedPermissions = addedPermissions
	return secureWorkflowReponse, nil
}
