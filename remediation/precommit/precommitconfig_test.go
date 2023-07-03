package precommit

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"testing"
)

func TestUpdatePrecommitConfig(t *testing.T) {

	const inputDirectory = "../../testfiles/precommit/input"
	const outputDirectory = "../../testfiles/precommit/output"

	tests := []struct {
		fileName  string
		Languages []string
		isChanged bool
	}{
		{
			fileName:  "basic.yml",
			Languages: []string{"JavaScript", "C++"},
			isChanged: true,
		},
		{
			fileName:  "file-not-exit.yml",
			Languages: []string{"JavaScript", "C++"},
			isChanged: true,
		},
		{
			fileName:  "same-repo-different-hooks.yml",
			Languages: []string{"Ruby", "Shell"},
			isChanged: true,
		},
		{
			fileName:  "style1.yml",
			Languages: []string{"Ruby", "Shell"},
			isChanged: true,
		},
	}

	for _, test := range tests {
		var updatePrecommitConfigRequest UpdatePrecommitConfigRequest
		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}
		updatePrecommitConfigRequest.Content = string(input)
		updatePrecommitConfigRequest.Languages = test.Languages
		inputRequest, err := json.Marshal(updatePrecommitConfigRequest)
		if err != nil {
			log.Fatal(err)
		}

		hooks, err := GetHooks(string(inputRequest))
		if err != nil {
			log.Fatal(err)
		}
		output, err := UpdatePrecommitConfig(string(inputRequest), hooks)
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
