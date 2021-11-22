package main

import (
	"io/ioutil"
	"log"
	"path"
	"testing"
)

func TestSecureWorkflow(t *testing.T) {
	const inputDirectory = "./testfiles/secureworkflow/input"
	const outputDirectory = "./testfiles/secureworkflow/output"
	files, err := ioutil.ReadDir(inputDirectory)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		output, err := SecureWorkflow(string(input), &mockDynamoDBClient{})

		if err != nil {
			t.Errorf("Error not expected")
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		if output.FinalOutput != string(expectedOutput) {
			t.Errorf("test failed %s did not match expected output\n%s", f.Name(), output.FinalOutput)
		}
	}

}
