package docker

import (
	"io/ioutil"
	"log"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

var resp = httpmock.File("../../testfiles/dockerfiles/response.json").String()

func TestSecureDockerFile(t *testing.T) {

	const inputDirectory = "../../testfiles/dockerfiles/input"
	const outputDirectory = "../../testfiles/dockerfiles/output"
	// NOTE: http mocking is not working,
	// need to investigate this issue
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// NOTE: below hack is required to capture docker api calls
	saveTr := Tr
	defer func() { Tr = saveTr }()
	Tr = httpmock.DefaultTransport

	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/library/python/manifests/3.7", httpmock.NewStringResponder(200, resp))

	tests := []struct {
		fileName  string
		isChanged bool
	}{
		{fileName: "Dockerfile-not-pinned", isChanged: true},
		{fileName: "Dockerfile-not-pinned-as", isChanged: true},
		{fileName: "Dockerfile-multiple-images", isChanged: true},
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

func Test_getSHA(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	saveTr := Tr
	defer func() { Tr = saveTr }()
	Tr = httpmock.DefaultTransport

	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/library/python/manifests/3.7", httpmock.NewStringResponder(200, resp))

	type args struct {
		image string
		tag   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "test1", args: args{image: "python", tag: "3.7"}, want: "sha256:5fb6f4b9d73ddeb0e431c938bee25c69157a1e3c880a81ff72c43a8055628de5", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSHA(tt.args.image, tt.args.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSHA() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getSHA() = %v, want %v", got, tt.want)
			}
		})
	}
}
