package main

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func AddAction(inputYaml, action string) (string, error) {
	workflow := Workflow{}
	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return "", fmt.Errorf("unable to parse yaml %v", err)
	}
	out := inputYaml
	currentJobLine := 0
	for jobName, job := range workflow.Jobs {
		alreadyPresent := false
		for _, step := range job.Steps {
			if len(step.Uses) > 0 && step.Uses == action {
				alreadyPresent = true
			}
		}

		if alreadyPresent {
			t := yaml.Node{}

			err := yaml.Unmarshal([]byte(inputYaml), &t)
			if err != nil {
				return "", fmt.Errorf("unable to parse yaml %v", err)
			}

			jobNode := iterateNode(&t, "steps", "!!seq", currentJobLine)
			currentJobLine = jobNode.Line
		} else {
			out, currentJobLine, err = addAction(out, jobName, action, currentJobLine)
		}

		if err != nil {
			return out, err
		}
	}

	return out, nil
}

func addAction(inputYaml, jobName, action string, minLine int) (string, int, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", 0, fmt.Errorf("unable to parse yaml %v", err)
	}

	jobNode := iterateNode(&t, "steps", "!!seq", minLine)

	if jobNode == nil {
		return "", 0, fmt.Errorf("jobName %s not found in the input yaml", jobName)
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

	output = append(output, spaces+fmt.Sprintf("- uses: %s", action))

	for i := jobNode.Line - 1; i < len(inputLines); i++ {
		output = append(output, inputLines[i])
	}

	return strings.Join(output, "\n"), jobNode.Line, nil
}
