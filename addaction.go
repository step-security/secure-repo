package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func AddAction(inputYaml, action string) (string, bool, error) {
	workflow := Workflow{}
	updated := false
	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return "", updated, fmt.Errorf("unable to parse yaml %v", err)
	}
	out := inputYaml

	for jobName, job := range workflow.Jobs {
		// Skip adding action for reusable jobs
		if IsCallingReusableWorkflow(job) {
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

	return out, updated, nil
}

func addAction(inputYaml, jobName, action string) (string, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", fmt.Errorf("unable to parse yaml %v", err)
	}

	jobNode := iterateNode(&t, jobName, "!!map", 0)

	jobNode = iterateNode(&t, "steps", "!!seq", jobNode.Line)

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
	output = append(output, spaces+"    egress-policy: audit # TODO: change to 'egress-policy: block' after couple of runs")
	output = append(output, "")

	for i := jobNode.Line - 1; i < len(inputLines); i++ {
		output = append(output, inputLines[i])
	}

	return strings.Join(output, "\n"), nil
}
