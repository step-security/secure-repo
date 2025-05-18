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
	HardenRunnerActionPath = "step-security/harden-runner"
	HardenRunnerActionName = "Harden the runner (Audit all outbound calls)"
)

func AddAction(inputYaml, action string, pinActions, pinToImmutable bool, skipContainerJobs bool) (string, bool, error) {
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
			out, err = addAction(out, jobName, action)
			if err != nil {
				return out, updated, err
			}
			updated = true
		}
	}

	if updated && pinActions {
		out, _ = pin.PinAction(action, out, nil, pinToImmutable)
	}

	return out, updated, nil
}

func addAction(inputYaml, jobName, action string) (string, error) {
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

	spaces := ""
	for i := 0; i < jobNode.Column-1; i++ {
		spaces += " "
	}

	output = append(output, spaces+fmt.Sprintf("- name: %s", HardenRunnerActionName))
	output = append(output, spaces+fmt.Sprintf("  uses: %s", action))
	output = append(output, spaces+"  with:")
	output = append(output, spaces+"    egress-policy: audit")
	output = append(output, "")

	for i := jobNode.Line - 1; i < len(inputLines); i++ {
		output = append(output, inputLines[i])
	}

	return strings.Join(output, "\n"), nil
}
