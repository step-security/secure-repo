package main

import (
	"io/ioutil"
	"log"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestSecureDockerFile(t *testing.T) {

	const inputDirectory = "./testfiles/dockerfiles/input"
	const outputDirectory = "./testfiles/dockerfiles/output"
	// NOTE: http mocking is not working,
	// need to investigate this issue
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	//Ping Docker Image
	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/",
		httpmock.NewStringResponder(200, `{
		}`))

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/",
		httpmock.NewStringResponder(200, `{
		}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/",
		httpmock.NewStringResponder(200, `{
		}`))

	httpmock.RegisterNoResponder(httpmock.NewStringResponder(500, `af`))

	tests := []struct {
		fileName  string
		isChanged bool
	}{
		{fileName: "Dockerfile-not-pinned", isChanged: true},
		{fileName: "Dockerfile-not-pinned-as", isChanged: true},
	}

	for _, test := range tests {

		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}

		output, err := SecureDockerFile(string(input))
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
