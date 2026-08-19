package maintainedactions

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/step-security/secure-repo/remediation/workflow/metadata"
	"github.com/step-security/secure-repo/remediation/workflow/permissions"
	"github.com/step-security/secure-repo/remediation/workflow/pin"
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

// resolveVersion determines the version to use for the replacement action.
// When replaceByMajorTag is true, it matches the major version from the original action.
// When false (default), it uses the latest release of the new action.
func resolveVersion(originalUses, actionName, newAction string, replaceByMajorTag bool) (string, error) {
	if !replaceByMajorTag {
		return GetLatestRelease(newAction)
	}

	parts := strings.SplitN(originalUses, "@", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", fmt.Errorf("no ref found in %s", originalUses)
	}
	ref := parts[1]
	pinnedBySHA := len(ref) == 40 && pin.IsAllHex(ref)

	// A tag ref states its own major version. A SHA carries none, so it has to be
	// resolved to the tag on that commit first.
	semanticVersion := ""
	majorVersion := getMajorVersion(ref)
	// above two are for the usual case

	if pinnedBySHA {
		var err error
		semanticVersion, err = tagForSHA(actionName, ref)
		if err != nil {
			return "", err
		}
		majorVersion = getMajorVersion(semanticVersion)
	}

	// The replacement is written as the major tag, so the fork must have it.
	// Checked before resolving the concrete version: if there is no matching
	// major to replace with, the version is irrelevant.
	forkMajorTag, exists, err := GetMajorTagIfExists(newAction, majorVersion)
	if err != nil || !exists {
		return "", fmt.Errorf("major tag %s not found on %s", majorVersion, newAction)
	}

	// Resolve the concrete version the original is actually pinned to. A major
	// tag such as "v4" is resolved through its commit SHA to the version it
	// currently points at (e.g. "v4.2.1").
	if !pinnedBySHA {
		semanticVersion = ref
		if !isConcreteSemver(ref) {
			// A major version such as "v5". Anything else without a minor version
			// (a branch name like "main", "latest", a short SHA) cannot reach this
			// point: the major-tag check above would not have found a matching tag
			// on the fork for it, so the replacement was already skipped.
			sha, err := GetSHAFromTag(actionName, ref)
			if isNotFound(err) {
				// No such tag. Some actions publish the floating major as a
				// branch instead, so try that before giving up. Any other
				// failure is passed through rather than retried as a branch.
				sha, err = GetSHAFromBranch(actionName, ref)
			}
			if err != nil {
				return "", fmt.Errorf("unable to resolve %s to a commit SHA as a tag or a branch: %w", ref, err)
			}
			semanticVersion, err = tagForSHA(actionName, sha)
			if err != nil {
				return "", err
			}
		}
	}

	// Maintained forks can lag behind the upstream action, so a matching major is
	// not enough: the fork's major tag may point at an older release than the one
	// the workflow is on, even within the same major. Compare against the version
	// the fork's major tag actually resolves to — that is what the workflow would
	// run — and skip the replacement when it is older. If that version cannot be
	// determined, skip as well rather than risk a downgrade.
	if isConcreteSemver(semanticVersion) {
		forkVersion, err := VersionForMajorTag(newAction, forkMajorTag)
		if err != nil {
			return "", fmt.Errorf("unable to determine which version %s@%s points at: %w", newAction, forkMajorTag, err)
		}
		if compareSemver(forkVersion, semanticVersion) < 0 {
			return "", fmt.Errorf("%s@%s is on %s, older than %s", newAction, forkMajorTag, forkVersion, semanticVersion)
		}
	}

	return forkMajorTag, nil
}

// ReplaceActions replaces original actions with Step Security actions in a workflow.
// When replaceByMajorTag is true, the replacement action uses the same major version as the original.
// When false (default), it uses the latest release of the replacement action.
func ReplaceActions(inputYaml string, customerMaintainedActions map[string]string, replaceByMajorTag bool) (string, bool, error) {
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
			actionName := strings.Split(step.Uses, "@")[0]
			if newAction, ok := actionMap[actionName]; ok {
				version, err := resolveVersion(step.Uses, actionName, newAction, replaceByMajorTag)
				if err != nil {
					log.Printf("skipping replacement of %s: %v", step.Uses, err)
					continue
				}
				replacements = append(replacements, replacement{
					jobName:        jobName,
					stepIdx:        stepIdx,
					newAction:      newAction,
					originalAction: step.Uses,
					latestVersion:  version,
				})
			}
		}
	}

	// For composite actions
	if workflow.Runs.Using == "composite" {
		for stepIdx, step := range workflow.Runs.Steps {
			if len(step.Uses) > 0 {
				actionName := strings.Split(step.Uses, "@")[0]
				if newAction, ok := actionMap[actionName]; ok {
					version, err := resolveVersion(step.Uses, actionName, newAction, replaceByMajorTag)
					if err != nil {
						log.Printf("skipping replacement of %s: %v", step.Uses, err)
						continue
					}
					replacements = append(replacements, replacement{
						jobName:        "composite",
						stepIdx:        stepIdx,
						newAction:      newAction,
						originalAction: step.Uses,
						latestVersion:  version,
					})
				}
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
		var stepsNode *yaml.Node

		if r.jobName == "composite" {
			// Handle composite actions
			runsNode := permissions.IterateNode(t, "runs", "!!map", 0)
			stepsNode = permissions.IterateNode(runsNode, "steps", "!!seq", 0)
		} else {
			// Handle regular workflow jobs
			jobsNode := permissions.IterateNode(t, "jobs", "!!map", 0)
			jobNode := permissions.IterateNode(jobsNode, r.jobName, "!!map", 0)
			stepsNode = permissions.IterateNode(jobNode, "steps", "!!seq", 0)
		}

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
