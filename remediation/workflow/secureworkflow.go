package workflow

import (
	"encoding/json"
	"log"

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
	enableLogging := false
	addEmptyTopLevelPermissions := false
	skipHardenRunnerForContainers := false
	exemptedActions, pinToImmutable, maintainedActionsMap := []string{}, false, map[string]string{}

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
			maintainedActionsMap = v
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

	if len(maintainedActionsMap) > 0 {
		replaceMaintainedActions = true
	}

	if queryStringParams["enableLogging"] == "true" {
		enableLogging = true
	}

	if queryStringParams["addEmptyTopLevelPermissions"] == "true" {
		addEmptyTopLevelPermissions = true
	}

	if queryStringParams["skipHardenRunnerForContainers"] == "true" {
		skipHardenRunnerForContainers = true
	}

	if enableLogging {
		// Log query parameters
		paramsJSON, _ := json.MarshalIndent(queryStringParams, "", "  ")
		log.Printf("SecureWorkflow called with query parameters: %s", paramsJSON)

		// Log input YAML (complete)
		log.Printf("Input YAML: %s", inputYaml)
	}

	secureWorkflowReponse := &permissions.SecureWorkflowReponse{FinalOutput: inputYaml, OriginalInput: inputYaml}
	var err error
	if addPermissions {
		if enableLogging {
			log.Printf("Adding job level permissions")
		}
		secureWorkflowReponse, err = permissions.AddJobLevelPermissions(secureWorkflowReponse.FinalOutput, addEmptyTopLevelPermissions)
		secureWorkflowReponse.OriginalInput = inputYaml
		if err != nil {
			if enableLogging {
				log.Printf("Error adding job level permissions: %v", err)
			}
			return nil, err
		} else {
			if !secureWorkflowReponse.HasErrors || permissions.ShouldAddWorkflowLevelPermissions(secureWorkflowReponse.JobErrors) {
				if enableLogging {
					log.Printf("Adding workflow level permissions")
				}
				secureWorkflowReponse.FinalOutput, err = permissions.AddWorkflowLevelPermissions(secureWorkflowReponse.FinalOutput, addProjectComment, addEmptyTopLevelPermissions)
				if err != nil {
					if enableLogging {
						log.Printf("Error adding workflow level permissions: %v", err)
					}
					secureWorkflowReponse.HasErrors = true
				} else {
					// reset the error
					// this is done because workflow perms have been added
					// only job errors were that perms already existed
					secureWorkflowReponse.HasErrors = false
				}
			}
			if len(secureWorkflowReponse.MissingActions) > 0 && !ignoreMissingKBs {
				if enableLogging {
					log.Printf("Storing missing actions: %v", secureWorkflowReponse.MissingActions)
				}
				StoreMissingActions(secureWorkflowReponse.MissingActions, svc)
			}
		}
		// if there are no errors, then we must have added perms
		// if there are already perms at workflow level, that is treated as an error condition
		addedPermissions = !secureWorkflowReponse.HasErrors
	}

	if replaceMaintainedActions {
		secureWorkflowReponse.FinalOutput, replacedMaintainedActions, err = maintainedactions.ReplaceActions(secureWorkflowReponse.FinalOutput, maintainedActionsMap)
		if err != nil {
			secureWorkflowReponse.HasErrors = true
		}
	}

	if pinActions {
		if enableLogging {
			log.Printf("Pinning GitHub Actions")
		}
		pinnedAction, pinnedDocker := false, false
		secureWorkflowReponse.FinalOutput, pinnedAction, _ = pin.PinActions(secureWorkflowReponse.FinalOutput, exemptedActions, pinToImmutable)
		secureWorkflowReponse.FinalOutput, pinnedDocker, _ = pin.PinDocker(secureWorkflowReponse.FinalOutput)
		pinnedActions = pinnedAction || pinnedDocker
		if enableLogging {
			log.Printf("Pinned actions: %v, Pinned docker: %v", pinnedAction, pinnedDocker)
		}
	}

	if addHardenRunner {
		if enableLogging {
			log.Printf("Adding harden runner action")
		}
		// Always pin harden-runner unless exempted
		pinHardenRunner := true
		if pin.ActionExists(HardenRunnerActionPath, exemptedActions) {
			pinHardenRunner = false
			if enableLogging {
				log.Printf("Harden runner action is exempted from pinning")
			}
		}
		secureWorkflowReponse.FinalOutput, addedHardenRunner, _ = hardenrunner.AddAction(secureWorkflowReponse.FinalOutput, HardenRunnerActionPathWithTag, pinHardenRunner, pinToImmutable, skipHardenRunnerForContainers)
		if enableLogging {
			log.Printf("Added harden runner: %v", addedHardenRunner)
		}
	}

	// Setting appropriate flags
	secureWorkflowReponse.PinnedActions = pinnedActions
	secureWorkflowReponse.AddedHardenRunner = addedHardenRunner
	secureWorkflowReponse.AddedPermissions = addedPermissions
	secureWorkflowReponse.AddedMaintainedActions = replacedMaintainedActions

	if enableLogging {
		log.Printf("SecureWorkflow complete - PinnedActions: %v, AddedHardenRunner: %v, AddedPermissions: %v, HasErrors: %v",
			secureWorkflowReponse.PinnedActions,
			secureWorkflowReponse.AddedHardenRunner,
			secureWorkflowReponse.AddedPermissions,
			secureWorkflowReponse.HasErrors)
	}

	return secureWorkflowReponse, nil
}
