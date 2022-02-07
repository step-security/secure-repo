package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"gopkg.in/yaml.v3"
)

func TestDockerActions(t *testing.T) {
	const inputDirectory = "./testfiles/pindockers/input"
	const outputDirectory = "./testfiles/pindockers/output"
	files, err := ioutil.ReadDir(inputDirectory)
	if err != nil {
		log.Fatal(err)
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://ghcr.io/token?scope=repository%3Astep-security%2Fintegration-test%2Fnt%3Apull&service=ghcr.io",
		httpmock.NewStringResponder(200, `{
			"token":"djE6c3RlcC1zZWN1cml0eS9pbnRlZ3JhdGlvbi10ZXN0L2ludDoxNjQ0MjI5Njc0MzY2ODE1NDA2"
		  }`))

	// add Table-Driven Tests

	for _, f := range files {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, f.Name()))

		if err != nil {
			fmt.Println("c1")
			log.Fatal(err)
		}

		output, err := PinDocker(string(input))

		if err != nil {
			t.Errorf("Error not expected")
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, f.Name()))

		if err != nil {
			fmt.Println("c2")
			log.Fatal(err)
		}

		if output != string(expectedOutput) {
			ioutil.WriteFile("out.yml", []byte(output), 0644)
			t.Errorf("test failed %s did not match expected output\n%s", f.Name(), output)
		}
	}
}
func PinDocker(inputYaml string) (string, error) {
	workflow := Workflow{}

	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return inputYaml, fmt.Errorf("unable to parse yaml %v", err)
	}

	out := inputYaml

	for jobName, job := range workflow.Jobs {

		for _, step := range job.Steps {
			if len(step.Uses) > 0 && strings.HasPrefix(step.Uses, "docker://") {
				out = pinDocker(step.Uses, jobName, out)
			}
		}
	}

	return out, nil
}
