package hardenrunner

import (
	"fmt"
	"log"
	"path"
	"strings"

	metadata "github.com/step-security/secure-repo/remediation/workflow/metadata"
	"github.com/step-security/secure-repo/remediation/workflow/permissions"
	"github.com/step-security/secure-repo/remediation/workflow/pin"
	"gopkg.in/yaml.v3"
)

const (
	HardenRunnerActionPath    = "step-security/harden-runner"
	HardenRunnerActionName    = "Harden the runner (Audit all outbound calls)"
	DefaultHardenRunnerConfig = "- name: Harden the runner (Audit all outbound calls)\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: audit"
)

type HardenRunnerConfig struct {
	Config           string   `json:"config"`
	Subtractive      bool     `json:"subtractive"`
	SkipHardenRunner bool     `json:"skipHardenRunner"`
	RunnerLabels     []string `json:"runnerLabels"`
	// ExemptRunnerLabels lists runner-label glob patterns (path.Match syntax, e.g.
	// "gpu-*"). When any of a job's runs-on labels matches any pattern, Harden-Runner
	// is NOT added to that job. This is an exclusion that takes precedence over the
	// RunnerLabels allow-list and applies regardless of SkipHardenRunner.
	ExemptRunnerLabels []string `json:"exemptRunnerLabels,omitempty"`
}

// getJobRunsOnLabels extracts the runs-on labels from a job's yaml.Node.
// Handles scalar (runs-on: ubuntu-latest), sequence (runs-on: [self-hosted, linux]),
// and mapping with labels key (runs-on: {labels: [self-hosted, linux]}) formats.
func getJobRunsOnLabels(jobNode *yaml.Node) []string {
	for i := 0; i < len(jobNode.Content); i += 2 {
		keyNode := jobNode.Content[i]
		if keyNode.Value == "runs-on" && i+1 < len(jobNode.Content) {
			return extractLabels(jobNode.Content[i+1])
		}
	}
	return nil
}

// extractLabels extracts labels from a yaml.Node that can be a scalar, sequence, or mapping with a "labels" key.
func extractLabels(node *yaml.Node) []string {
	switch node.Kind {
	case yaml.ScalarNode:
		return []string{node.Value}
	case yaml.SequenceNode:
		var labels []string
		for _, item := range node.Content {
			labels = append(labels, item.Value)
		}
		return labels
	case yaml.MappingNode:
		for j := 0; j < len(node.Content); j += 2 {
			if node.Content[j].Value == "labels" && j+1 < len(node.Content) {
				return extractLabels(node.Content[j+1])
			}
		}
	}
	return nil
}

// shouldSkipJob returns true if none of the job's runs-on labels match the allowed labels.
func shouldSkipJob(jobLabels []string, allowedLabels []string) bool {
	for _, jl := range jobLabels {
		for _, al := range allowedLabels {
			// TODO CHECK CASE INSENSITIVE MATCHING
			if jl == al {
				return false
			}
		}
	}
	return true
}

// isExemptJob returns true if any of the job's runs-on labels matches any of the
// exempt patterns. Patterns use path.Match glob syntax (e.g. "gpu-*", "arm64-?");
// a pattern with no wildcard is an exact match. Matching is case-insensitive. Used
// to skip adding Harden-Runner to jobs whose runner is explicitly exempted.
func isExemptJob(jobLabels []string, exemptPatterns []string) bool {
	for _, jl := range jobLabels {
		for _, pat := range exemptPatterns {
			if matched, err := path.Match(strings.ToLower(pat), strings.ToLower(jl)); err == nil && matched {
				return true
			}
		}
	}
	return false
}

// getActionFromConfig parses the "uses:" line from the Config yaml string.
// Falls back to HardenRunnerActionPath if no uses line is present.
func getActionFromConfig(config HardenRunnerConfig) string {
	for _, line := range strings.Split(config.Config, "\n") {
		trimmed := strings.TrimPrefix(strings.TrimSpace(line), "- ")
		if strings.HasPrefix(trimmed, "uses:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "uses:"))
		}
	}
	return HardenRunnerActionPath
}

// getJobStepsNode locates the block-style steps sequence for jobName by walking
// the jobs mapping directly. Line-based lookups (permissions.IterateNode) mis-resolve
// YAML anchors and aliases: an aliased `steps: *anchor` node carries no !!seq tag,
// so a line-based search skips it and matches a different job's steps sequence,
// which corrupts the workflow when lines are spliced in at that position.
// Returns nil when the job's steps cannot be safely edited by line splicing:
//   - steps is an alias (*anchor) — the anchor job owns the shared content
//   - steps is flow-style ([...]) or empty
//   - the job itself is an alias, or the job/steps key is missing
func getJobStepsNode(root *yaml.Node, jobName string) *yaml.Node {
	jobsNode := permissions.IterateNode(root, "jobs", "!!map", 0)
	if jobsNode == nil {
		return nil
	}
	for i := 0; i+1 < len(jobsNode.Content); i += 2 {
		if jobsNode.Content[i].Value != jobName {
			continue
		}
		jobValue := jobsNode.Content[i+1]
		if jobValue.Kind != yaml.MappingNode {
			return nil
		}
		for j := 0; j+1 < len(jobValue.Content); j += 2 {
			if jobValue.Content[j].Value != "steps" {
				continue
			}
			stepsNode := jobValue.Content[j+1]
			if stepsNode.Kind != yaml.SequenceNode || stepsNode.Style == yaml.FlowStyle || len(stepsNode.Content) == 0 {
				return nil
			}
			return stepsNode
		}
		return nil
	}
	return nil
}

// leadingWhitespace returns the leading spaces/tabs of s.
func leadingWhitespace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}

func AddAction(inputYaml string, hardenRunnerConfig HardenRunnerConfig, pinActions, pinToImmutable bool, skipContainerJobs bool) (string, bool, error) {
	if hardenRunnerConfig.Config == "" {
		hardenRunnerConfig.Config = DefaultHardenRunnerConfig
	}
	workflow := metadata.Workflow{}
	updated := false
	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return inputYaml, updated, fmt.Errorf("unable to parse yaml %v", err)
	}

	// Extract the action path from the config to detect custom actions already present.
	configAction := getActionFromConfig(hardenRunnerConfig)
	configActionPath := strings.Split(configAction, "@")[0]

	// Build a map of jobName → yaml.Node for runs-on label lookup. Needed for the
	// RunnerLabels allow-list and/or the ExemptRunnerLabels exclusion.
	jobNodeMap := map[string]*yaml.Node{}
	if (hardenRunnerConfig.SkipHardenRunner && len(hardenRunnerConfig.RunnerLabels) > 0) || len(hardenRunnerConfig.ExemptRunnerLabels) > 0 {
		t := yaml.Node{}
		if err := yaml.Unmarshal([]byte(inputYaml), &t); err == nil {
			jobsNode := permissions.IterateNode(&t, "jobs", "!!map", 0)
			if jobsNode != nil {
				for i := 0; i < len(jobsNode.Content); i += 2 {
					jobNodeMap[jobsNode.Content[i].Value] = jobsNode.Content[i+1]
				}
			}
		}
	}

	out := inputYaml

	for jobName, job := range workflow.Jobs {
		// Skip adding action for reusable jobs
		if metadata.IsCallingReusableWorkflow(job) {
			continue
		}
		// Skip adding action for jobs running in containers if skipContainerJobs is true
		if skipContainerJobs && job.Container.Image != "" {
			continue
		}
		// Skip jobs whose runner is exempted. This is an exclusion (glob match on the
		// job's runs-on labels), takes precedence over the RunnerLabels allow-list, and
		// applies regardless of SkipHardenRunner.
		if len(hardenRunnerConfig.ExemptRunnerLabels) > 0 {
			if jn, ok := jobNodeMap[jobName]; ok {
				if isExemptJob(getJobRunsOnLabels(jn), hardenRunnerConfig.ExemptRunnerLabels) {
					continue
				}
			}
		}
		// Skip jobs whose runs-on label doesn't match the allowed labels
		if hardenRunnerConfig.SkipHardenRunner && len(hardenRunnerConfig.RunnerLabels) > 0 {
			if jn, ok := jobNodeMap[jobName]; ok {
				if shouldSkipJob(getJobRunsOnLabels(jn), hardenRunnerConfig.RunnerLabels) {
					continue
				}
			}
		}
		alreadyPresent := false
		for _, step := range job.Steps {
			if len(step.Uses) > 0 && (strings.HasPrefix(step.Uses, HardenRunnerActionPath) || strings.HasPrefix(step.Uses, configActionPath)) {
				alreadyPresent = true
				break
			}
		}

		if !alreadyPresent {
			var changed bool
			out, changed, err = addAction(out, jobName, hardenRunnerConfig)
			if err != nil {
				return out, updated, err
			}
			if changed {
				updated = true
			}
		} else if hardenRunnerConfig.Subtractive {
			var changed bool
			out, changed, err = updateHardenRunnerConfig(out, jobName, hardenRunnerConfig)
			if err != nil {
				return out, updated, err
			}
			if changed {
				updated = true
			}
		}
	}

	if updated && pinActions {
		action := getActionFromConfig(hardenRunnerConfig)
		pinnedOut, _, pinErr := pin.PinActionWithPatFallback(action, out, nil, pinToImmutable, nil)
		if pinErr != nil {
			// Non-fatal: keep the unpinned harden-runner step rather than dropping
			// the addition entirely (matches previous net behavior, where this
			// error was discarded by the caller).
			log.Printf("unable to pin harden runner action, keeping unpinned: %v", pinErr)
		} else {
			out = pinnedOut
		}
	}

	if out == inputYaml {
		updated = false
	}

	return out, updated, nil
}

func hardenRunnerConfigMatches(inputLines []string, hrStartLine, hrEndLine int, spaces, config, existingTagOrSHA string) bool {
	var newConfigLines []string
	for _, line := range strings.Split(config, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if existingTagOrSHA != "" {
			check := strings.TrimPrefix(strings.TrimSpace(line), "- ")
			if strings.HasPrefix(check, "uses:") {
				usesValue := strings.TrimSpace(strings.TrimPrefix(check, "uses:"))
				if idx := strings.Index(usesValue, "@"); idx >= 0 {
					line = strings.Replace(line, usesValue, usesValue[:idx]+"@"+existingTagOrSHA, 1)
				}
			}
		}
		newConfigLines = append(newConfigLines, spaces+line)
	}
	newConfigLines = append(newConfigLines, "")
	return strings.Join(inputLines[hrStartLine:hrEndLine], "\n") == strings.Join(newConfigLines, "\n")
}

// getHardenRunnerStepLines locates the harden-runner (or custom action) step in the job
// and returns its start/end line indices, indentation spaces, existing action path, and existing tag/SHA.
func getHardenRunnerStepLines(inputYaml, jobName, configActionPath string) (hrStartLine, hrEndLine int, spaces, existingActionPath, existingTagOrSHA string, err error) {
	t := yaml.Node{}
	if err = yaml.Unmarshal([]byte(inputYaml), &t); err != nil {
		return -1, -1, "", "", "", fmt.Errorf("unable to parse yaml %v", err)
	}

	stepsNode := getJobStepsNode(&t, jobName)
	if stepsNode == nil {
		// Steps cannot be safely edited by line splicing (alias, flow style,
		// or missing); report "not found" so the caller leaves the job as is.
		return -1, -1, "", "", "", nil
	}

	inputLines := strings.Split(inputYaml, "\n")
	// Indentation from the first step's own line is anchor-safe: for
	// `steps: &anchor` the sequence node reports the anchor's position on the
	// `steps:` line itself, not the first step.
	spaces = leadingWhitespace(inputLines[stepsNode.Content[0].Line-1])
	hrStartLine = -1
	hrEndLine = len(inputLines)

	for i, stepNode := range stepsNode.Content {
		matched := false
		for j := 0; j+1 < len(stepNode.Content); j += 2 {
			if stepNode.Content[j].Value == "uses" {
				usesVal := stepNode.Content[j+1].Value
				if strings.HasPrefix(usesVal, HardenRunnerActionPath) || strings.HasPrefix(usesVal, configActionPath) {
					matched = true
					break
				}
			}
		}
		if !matched {
			continue
		}

		hrStartLine = stepNode.Line - 1

		// extract existing action path and tag/SHA from the raw text
		for _, rawLine := range inputLines[hrStartLine:] {
			trimmed := strings.TrimPrefix(strings.TrimSpace(rawLine), "- ")
			if strings.HasPrefix(trimmed, "uses:") {
				usesValue := strings.TrimSpace(strings.TrimPrefix(trimmed, "uses:"))
				if idx := strings.Index(usesValue, "@"); idx >= 0 {
					existingActionPath = usesValue[:idx]
					existingTagOrSHA = usesValue[idx+1:]
				} else {
					existingActionPath = usesValue
				}
				break
			}
		}

		// compute hrEndLine
		if i+1 < len(stepsNode.Content) {
			hrEndLine = stepsNode.Content[i+1].Line - 1
		} else {
			// last step — scan forward until a line is no longer part of this step
			stepContentPrefix := spaces + " "
			for j := hrStartLine + 1; j < len(inputLines); j++ {
				line := inputLines[j]
				if strings.TrimSpace(line) == "" {
					continue
				}
				if !strings.HasPrefix(line, stepContentPrefix) {
					hrEndLine = j
					break
				}
			}
		}
		break
	}

	return hrStartLine, hrEndLine, spaces, existingActionPath, existingTagOrSHA, nil
}

func updateHardenRunnerConfig(inputYaml, jobName string, hardenRunnerConfig HardenRunnerConfig) (string, bool, error) {
	configAction := getActionFromConfig(hardenRunnerConfig)
	configActionPath := strings.Split(configAction, "@")[0]

	hrStartLine, hrEndLine, spaces, existingActionPath, existingTagOrSHA, err := getHardenRunnerStepLines(inputYaml, jobName, configActionPath)
	if err != nil {
		return inputYaml, false, err
	}
	if hrStartLine < 0 {
		return inputYaml, false, nil
	}

	// preserve tag/SHA only when the action path is unchanged
	tagToUse := ""
	if existingActionPath == configActionPath {
		tagToUse = existingTagOrSHA
	}

	inputLines := strings.Split(inputYaml, "\n")

	// already up to date — nothing to do
	if hardenRunnerConfigMatches(inputLines, hrStartLine, hrEndLine, spaces, hardenRunnerConfig.Config, tagToUse) {
		return inputYaml, false, nil
	}

	// rebuild: lines before + new config (with tag grafted if same path) + lines after
	var output []string
	output = append(output, inputLines[:hrStartLine]...)
	for _, line := range strings.Split(hardenRunnerConfig.Config, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if tagToUse != "" {
			check := strings.TrimPrefix(strings.TrimSpace(line), "- ")
			if strings.HasPrefix(check, "uses:") {
				usesValue := strings.TrimSpace(strings.TrimPrefix(check, "uses:"))
				if idx := strings.Index(usesValue, "@"); idx >= 0 {
					line = strings.Replace(line, usesValue, usesValue[:idx]+"@"+tagToUse, 1)
				}
			}
		}
		output = append(output, spaces+line)
	}
	output = append(output, "")
	output = append(output, inputLines[hrEndLine:]...)

	return strings.Join(output, "\n"), true, nil
}

func addAction(inputYaml, jobName string, hardenRunnerConfig HardenRunnerConfig) (string, bool, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return inputYaml, false, fmt.Errorf("unable to parse yaml %v", err)
	}

	stepsNode := getJobStepsNode(&t, jobName)
	if stepsNode == nil {
		// Steps cannot be safely edited by line splicing (e.g. defined via a
		// YAML alias, flow style, or missing); leave the job unchanged rather
		// than risk corrupting the workflow. Alias jobs inherit the step from
		// their anchor job.
		return inputYaml, false, nil
	}

	// Insert immediately before the first step. The first step's own line is
	// anchor-safe: for `steps: &anchor` the sequence node reports the anchor's
	// position on the `steps:` line itself, not the first step.
	insertLine := stepsNode.Content[0].Line // 1-based
	inputLines := strings.Split(inputYaml, "\n")
	if insertLine-1 >= len(inputLines) {
		return inputYaml, false, fmt.Errorf("steps for job %s out of line range", jobName)
	}

	var output []string
	output = append(output, inputLines[:insertLine-1]...)

	spaces := leadingWhitespace(inputLines[insertLine-1])

	for _, line := range strings.Split(hardenRunnerConfig.Config, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		output = append(output, spaces+line)
	}
	output = append(output, "")

	output = append(output, inputLines[insertLine-1:]...)

	return strings.Join(output, "\n"), true, nil
}
