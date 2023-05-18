package dependabot

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"testing"
)

func TestConfigDependabotFile(t *testing.T) {

	const inputDirectory = "../../testfiles/dependabotfiles/input"
	const outputDirectory = "../../testfiles/dependabotfiles/output"

	tests := []struct {
		fileName   string
		Ecosystems []Ecosystem
		isChanged  bool
	}{
		{
			fileName:   "Without-github-action.yml",
			Ecosystems: []Ecosystem{{"github-actions", "/", "daily"}, {"npm", "/app", "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "With-github-action.yml",
			Ecosystems: []Ecosystem{{"github-actions", "/", "daily"}},
			isChanged:  false,
		},
		{
			fileName:   "File-not-exit.yml",
			Ecosystems: []Ecosystem{{"github-actions", "/", "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "Same-ecosystem-different-directory.yml",
			Ecosystems: []Ecosystem{{"github-actions", "/", "daily"}, {"npm", "/sample", "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "No-Indentation.yml",
			Ecosystems: []Ecosystem{{"npm", "/sample", "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "High-Indentation.yml",
			Ecosystems: []Ecosystem{{"npm", "/sample", "daily"}},
			isChanged:  true,
		},
	}

	for _, test := range tests {
		var updateDependabotConfigRequest UpdateDependabotConfigRequest
		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}
		updateDependabotConfigRequest.Content = string(input)
		updateDependabotConfigRequest.Ecosystems = test.Ecosystems
		inputRequest, err := json.Marshal(updateDependabotConfigRequest)
		if err != nil {
			log.Fatal(err)
		}

		output, err := UpdateDependabotConfig(string(inputRequest))
		if err != nil {
			t.Fatalf("Error not expected: %s", err)
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}

		if string(expectedOutput) != output.FinalOutput {
			t.Errorf("test failed %s did not match expected output\n%s", test.fileName, output.FinalOutput)
		}

		if output.IsChanged != test.isChanged {
			t.Errorf("test failed %s did not match IsChanged, Expected: %v Got: %v", test.fileName, test.isChanged, output.IsChanged)

		}

	}

}
