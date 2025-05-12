package workflow

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/step-security/secure-repo/remediation/workflow/hardenrunner"
	"github.com/step-security/secure-repo/remediation/workflow/maintainedactions"
	"github.com/step-security/secure-repo/remediation/workflow/permissions"
	"github.com/step-security/secure-repo/remediation/workflow/pin"
)

const (
	HardenRunnerActionPathWithTag = "step-security/harden-runner@v2"
	HardenRunnerActionPath        = "step-security/harden-runner"
	HardenRunnerActionName        = "Harden Runner"
)

func SecureWorkflow(queryStringParams map[string]string, inputYaml string, svc dynamodbiface.DynamoDBAPI, params ...interface{}) (*permissions.SecureWorkflowReponse, error) {
	pinActions, addHardenRunner, addPermissions, addProjectComment, replaceMaintainedActions := true, true, true, true, false
	pinnedActions, addedHardenRunner, addedPermissions, replacedMaintainedActions := false, false, false, false
	ignoreMissingKBs := false
	exemptedActions, pinToImmutable, customerMaintainedActions := []string{}, false, map[string]string{}
	if len(params) > 0 {
		if v, ok := params[0].([]string); ok {
			exemptedActions = v
		}
	}
	if len(params) > 1 {
		if v, ok := params[1].(bool); ok {
			pinToImmutable = v
		}
	}
	if len(params) > 2 {
		if v, ok := params[2].(map[string]string); ok {
			customerMaintainedActions = v
		}
	}

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

	if len(customerMaintainedActions) > 0 {
		replaceMaintainedActions = true
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

	if replaceMaintainedActions {
		secureWorkflowReponse.FinalOutput, replacedMaintainedActions, err = maintainedactions.ReplaceActions(secureWorkflowReponse.FinalOutput, customerMaintainedActions)
		if err != nil {
			secureWorkflowReponse.HasErrors = true
		}
	}

	if pinActions {
		pinnedAction, pinnedDocker := false, false
		secureWorkflowReponse.FinalOutput, pinnedAction, _ = pin.PinActions(secureWorkflowReponse.FinalOutput, exemptedActions, pinToImmutable)
		secureWorkflowReponse.FinalOutput, pinnedDocker, _ = pin.PinDocker(secureWorkflowReponse.FinalOutput)
		pinnedActions = pinnedAction || pinnedDocker
	}

	if addHardenRunner {
		// Always pin harden-runner unless exempted
		pinHardenRunner := true
		if pin.ActionExists(HardenRunnerActionPath, exemptedActions) {
			pinHardenRunner = false
		}
		secureWorkflowReponse.FinalOutput, addedHardenRunner, _ = hardenrunner.AddAction(secureWorkflowReponse.FinalOutput, HardenRunnerActionPathWithTag, pinHardenRunner, pinToImmutable)
	}

	// Setting appropriate flags
	secureWorkflowReponse.PinnedActions = pinnedActions
	secureWorkflowReponse.AddedHardenRunner = addedHardenRunner
	secureWorkflowReponse.AddedPermissions = addedPermissions
	secureWorkflowReponse.AddedMaintainedActions = replacedMaintainedActions
	return secureWorkflowReponse, nil
}
