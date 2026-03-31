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
	Config      string
	Subtractive bool
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
		alreadyPresent := false
		for _, step := range job.Steps {
			if len(step.Uses) > 0 && strings.HasPrefix(step.Uses, HardenRunnerActionPath) {
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
