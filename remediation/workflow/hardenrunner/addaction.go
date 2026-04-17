package hardenrunner

import (
	"fmt"
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

// getActionFromConfig parses the "uses:" line from the Config yaml string.
// Falls back to HardenRunnerActionPath if no uses line is present.
func getActionFromConfig(config HardenRunnerConfig) string {
	for _, line := range strings.Split(config.Config, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "uses:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "uses:"))
		}
	}
	return HardenRunnerActionPath
}

func AddAction(inputYaml string, hardenRunnerConfig HardenRunnerConfig, pinActions, pinToImmutable bool, skipContainerJobs bool) (string, bool, error) {
	if hardenRunnerConfig.Config == "" {
		hardenRunnerConfig.Config = DefaultHardenRunnerConfig
	}
	workflow := metadata.Workflow{}
	updated := false
	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return "", updated, fmt.Errorf("unable to parse yaml %v", err)
	}

	// Extract the action path from the config to detect custom actions already present.
	configAction := getActionFromConfig(hardenRunnerConfig)
	configActionPath := strings.Split(configAction, "@")[0]

	// Build a map of jobName → yaml.Node for runs-on label lookup
	jobNodeMap := map[string]*yaml.Node{}
	if hardenRunnerConfig.SkipHardenRunner && len(hardenRunnerConfig.RunnerLabels) > 0 {
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
			out, err = addAction(out, jobName, hardenRunnerConfig)
			if err != nil {
				return out, updated, err
			}
			updated = true
		} else if hardenRunnerConfig.Subtractive {
			out, err = updateHardenRunnerConfig(out, jobName, hardenRunnerConfig)
			if err != nil {
				return out, updated, err
			}
			updated = true
		}
	}

	if updated && pinActions {
		action := getActionFromConfig(hardenRunnerConfig)
		out, _, err = pin.PinActionWithPatFallback(action, out, nil, pinToImmutable, nil)
		if err != nil {
			return out, updated, err
		}
	}

	return out, updated, nil
}

func updateHardenRunnerConfig(inputYaml, jobName string, hardenRunnerConfig HardenRunnerConfig) (string, error) {
	t := yaml.Node{}
	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", fmt.Errorf("unable to parse yaml %v", err)
	}

	jobNode := permissions.IterateNode(&t, "jobs", "!!map", 0)
	jobNode = permissions.IterateNode(&t, jobName, "!!map", jobNode.Line)
	stepsNode := permissions.IterateNode(&t, "steps", "!!seq", jobNode.Line)
	if stepsNode == nil {
		return "", fmt.Errorf("steps not found for job %s", jobName)
	}

	spaces := strings.Repeat(" ", stepsNode.Column-1)
	inputLines := strings.Split(inputYaml, "\n")

	hrStartLine := -1
	hrEndLine := len(inputLines)

	for i, stepNode := range stepsNode.Content {
		isHR := false
		for j := 0; j+1 < len(stepNode.Content); j += 2 {
			if stepNode.Content[j].Value == "uses" && strings.HasPrefix(stepNode.Content[j+1].Value, HardenRunnerActionPath) {
				isHR = true
				break
			}
		}
		if !isHR {
			continue
		}
		hrStartLine = stepNode.Line - 1 // convert to 0-indexed
		if i+1 < len(stepsNode.Content) {
			hrEndLine = stepsNode.Content[i+1].Line - 1
		} else {
			// last step — scan forward until line is no longer part of this step
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

	if hrStartLine < 0 {
		return inputYaml, nil
	}

	var output []string
	output = append(output, inputLines[:hrStartLine]...)
	for _, line := range strings.Split(hardenRunnerConfig.Config, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		output = append(output, spaces+line)
	}
	output = append(output, "")
	output = append(output, inputLines[hrEndLine:]...)

	return strings.Join(output, "\n"), nil
}

func addAction(inputYaml, jobName string, hardenRunnerConfig HardenRunnerConfig) (string, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", fmt.Errorf("unable to parse yaml %v", err)
	}

	jobNode := permissions.IterateNode(&t, "jobs", "!!map", 0)

	jobNode = permissions.IterateNode(&t, jobName, "!!map", jobNode.Line)

	jobNode = permissions.IterateNode(&t, "steps", "!!seq", jobNode.Line)

	if jobNode == nil {
		return "", fmt.Errorf("jobName %s not found in the input yaml", jobName)
	}

	inputLines := strings.Split(inputYaml, "\n")
	var output []string
	for i := 0; i < jobNode.Line-1; i++ {
		output = append(output, inputLines[i])
	}

	spaces := strings.Repeat(" ", jobNode.Column-1)

	for _, line := range strings.Split(hardenRunnerConfig.Config, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		output = append(output, spaces+line)
	}
	output = append(output, "")

	for i := jobNode.Line - 1; i < len(inputLines); i++ {
		output = append(output, inputLines[i])
	}

	return strings.Join(output, "\n"), nil
}
