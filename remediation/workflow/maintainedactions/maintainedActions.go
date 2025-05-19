package maintainedactions

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/step-security/secure-repo/remediation/workflow/metadata"
	"github.com/step-security/secure-repo/remediation/workflow/permissions"
	"gopkg.in/yaml.v3"
)

// Action represents a GitHub Action in the maintained actions list
type Action struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ForkedFrom  struct {
		Name string `json:"name"`
	} `json:"forkedFrom"`
	Score int    `json:"score"`
	Image string `json:"image"`
}

type replacement struct {
	jobName        string
	stepIdx        int
	newAction      string
	originalAction string
	latestVersion  string
}

// LoadMaintainedActions loads the maintained actions from the JSON file
func LoadMaintainedActions(jsonPath string) (map[string]string, error) {
	// Read the JSON file
	data, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read maintained actions file: %v", err)
	}

	// Parse the JSON
	var actions []Action
	if err := json.Unmarshal(data, &actions); err != nil {
		return nil, fmt.Errorf("failed to parse maintained actions JSON: %v", err)
	}

	// Create a map of original actions to their Step Security replacements
	actionMap := make(map[string]string)
	for _, action := range actions {
		if action.ForkedFrom.Name != "" {
			actionMap[action.ForkedFrom.Name] = action.Name
		}
	}

	return actionMap, nil
}

// ReplaceActions replaces original actions with Step Security actions in a workflow
func ReplaceActions(inputYaml string, customerMaintainedActions map[string]string) (string, bool, error) {
	workflow := metadata.Workflow{}
	updated := false

	actionMap := customerMaintainedActions

	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return "", updated, fmt.Errorf("unable to parse yaml: %v", err)
	}

	// Step 1: Check if anything needs to be replaced

	var replacements []replacement

	for jobName, job := range workflow.Jobs {
		if metadata.IsCallingReusableWorkflow(job) {
			continue
		}
		for stepIdx, step := range job.Steps {
			// fmt.Println("step ", step.Uses)
			actionName := strings.Split(step.Uses, "@")[0]
			if newAction, ok := actionMap[actionName]; ok {
				latestVersion, err := GetLatestRelease(newAction)
				if err != nil {
					return "", updated, fmt.Errorf("unable to get latest release: %v", err)
				}
				replacements = append(replacements, replacement{
					jobName:        jobName,
					stepIdx:        stepIdx,
					newAction:      newAction,
					originalAction: step.Uses,
					latestVersion:  latestVersion,
				})
			}
		}
	}
	if len(replacements) == 0 {
		// No changes needed
		return inputYaml, false, nil
	}

	// Step 2: Now modify the YAML lines manually
	t := yaml.Node{}
	err = yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", updated, fmt.Errorf("unable to parse yaml: %v", err)
	}

	inputLines := strings.Split(inputYaml, "\n")
	inputLines, updated = replaceAction(&t, inputLines, replacements, updated)

	output := strings.Join(inputLines, "\n")

	return output, updated, nil
}

func replaceAction(t *yaml.Node, inputLines []string, replacements []replacement, updated bool) ([]string, bool) {
	for _, r := range replacements {
		jobsNode := permissions.IterateNode(t, "jobs", "!!map", 0)
		jobNode := permissions.IterateNode(jobsNode, r.jobName, "!!map", 0)
		stepsNode := permissions.IterateNode(jobNode, "steps", "!!seq", 0)
		if stepsNode == nil {
			continue
		}

		// Now get the specific step
		stepNode := stepsNode.Content[r.stepIdx]
		usesNode := permissions.IterateNode(stepNode, "uses", "!!str", 0)
		if usesNode == nil {
			continue
		}

		lineNum := usesNode.Line - 1 // 0-based indexing
		columnNum := usesNode.Column - 1

		// Replace the line
		oldLine := inputLines[lineNum]
		prefix := oldLine[:columnNum]
		inputLines[lineNum] = prefix + r.newAction + "@" + r.latestVersion
		updated = true

	}
	return inputLines, updated
}
