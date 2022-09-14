package main

import (
	"io/ioutil"
	"log"
	"path"
	"testing"
)

func TestConfigDependabotFile(t *testing.T) {

	const inputDirectory = "./testfiles/dependabotfiles/input"
	const outputDirectory = "./testfiles/dependabotfiles/output"

	tests := []struct {
		fileName  string
		isChanged bool
	}{
		{fileName: "DependabotFile-without-github-action.yml", isChanged: true},
		{fileName: "DependabotFile-with-github-action.yml", isChanged: false},
	}

	for _, test := range tests {

		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}

		output, err := UpdateDependabotConfig(string(input))
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
