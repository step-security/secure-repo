package main

import (
	"io/ioutil"
	"log"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestPinActions(t *testing.T) {
	const inputDirectory = "./testfiles/pinactions/input"
	const outputDirectory = "./testfiles/pinactions/output"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/peter-evans/close-issue/commits/v1",
		httpmock.NewStringResponder(200, `a700eac5bf2a1c7a8cb6da0c13f93ed96fd53dbe`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/commits/master",
		httpmock.NewStringResponder(200, `61b9e3751b92087fd0b06925ba6dd6314e06f089`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/elgohr/Publish-Docker-Github-Action/commits/master",
		httpmock.NewStringResponder(200, `8217e91c0369a5342a4ef2d612de87492410a666`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/borales/actions-yarn/commits/v2.3.0",
		httpmock.NewStringResponder(200, `4965e1a0f0ae9c422a9a5748ebd1fb5e097d22b9`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/commits/v1",
		httpmock.NewStringResponder(200, `544eadc6bf3d226fd7a7a9f0dc5b5bf7ca0675b9`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/setup-node/commits/v1",
		httpmock.NewStringResponder(200, `f1f314fca9dfce2769ece7d933488f076716723e`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/JS-DevTools/npm-publish/commits/v1",
		httpmock.NewStringResponder(200, `0f451a94170d1699fd50710966d48fb26194d939`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/brandedoutcast/publish-nuget/commits/v2",
		httpmock.NewStringResponder(200, `c12b8546b67672ee38ac87bea491ac94a587f7cc`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/rohith/publish-nuget/commits/v2",
		httpmock.NewStringResponder(200, `c12b8546b67672ee38ac87bea491ac94a587f7cc`))

	tests := []struct {
		fileName    string
		wantUpdated bool
	}{
		{fileName: "alreadypinned.yml", wantUpdated: false},
		{fileName: "branch.yml", wantUpdated: true},
		{fileName: "localaction.yml", wantUpdated: true},
		{fileName: "multiplejobs.yml", wantUpdated: true},
		{fileName: "basic.yml", wantUpdated: true},
		{fileName: "dockeraction.yml", wantUpdated: true},
		{fileName: "multipleactions.yml", wantUpdated: true},
	}
	for _, tt := range tests {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.fileName))

		if err != nil {
			log.Fatal(err)
		}

		output, gotUpdated, err := PinActions(string(input))
		if tt.wantUpdated != gotUpdated {
			t.Errorf("test failed wantUpdated %v did not match gotUpdated %v", tt.wantUpdated, gotUpdated)
		}
		if err != nil {
			t.Errorf("Error not expected")
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, tt.fileName))

		if err != nil {
			log.Fatal(err)
		}

		if output != string(expectedOutput) {
			t.Errorf("test failed %s did not match expected output\n%s", tt.fileName, output)
		}
	}
}

func Test_isAbsolute(t *testing.T) {
	type args struct {
		ref string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{name: "Fixed Action", args: args{ref: "actions/setup-node@f1f314fca9dfce2769ece7d933488f076716723e"}, want: true},
		{name: "Unfixed Action", args: args{ref: "actions/setup-node@v1"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAbsolute(tt.args.ref); got != tt.want {
				t.Errorf("isAbsolute() = %v, want %v", got, tt.want)
			}
		})
	}
}
