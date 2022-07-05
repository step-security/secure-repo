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

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/peter-evans/close-issue/git/ref/tags/v1",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v1",
			"node_id": "MDM6UmVmMjYxMDY4MTU5OnJlZnMvdGFncy92MQ==",
			"url": "https://api.github.com/repos/peter-evans/close-issue/git/refs/tags/v1",
			"object": {
			  "sha": "a700eac5bf2a1c7a8cb6da0c13f93ed96fd53dbe",
			  "type": "commit",
			  "url": "https://api.github.com/repos/peter-evans/close-issue/git/commits/a700eac5bf2a1c7a8cb6da0c13f93ed96fd53dbe"
			}
		  }`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/git/ref/heads/master",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/heads/master",
			"node_id": "MDM6UmVmMTk3ODE0NjI5OnJlZnMvaGVhZHMvbWFzdGVy",
			"url": "https://api.github.com/repos/actions/checkout/git/refs/heads/master",
			"object": {
			  "sha": "61b9e3751b92087fd0b06925ba6dd6314e06f089",
			  "type": "commit",
			  "url": "https://api.github.com/repos/actions/checkout/git/commits/61b9e3751b92087fd0b06925ba6dd6314e06f089"
			}
		  }`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/elgohr/Publish-Docker-Github-Action/git/ref/heads/master",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/heads/master",
			"node_id": "MDM6UmVmMTcyMjA5NzEzOnJlZnMvaGVhZHMvbWFzdGVy",
			"url": "https://api.github.com/repos/elgohr/Publish-Docker-Github-Action/git/refs/heads/master",
			"object": {
			  "sha": "49d9c2e46838527972659f80f5488d08971fdc2d",
			  "type": "commit",
			  "url": "https://api.github.com/repos/elgohr/Publish-Docker-Github-Action/git/commits/49d9c2e46838527972659f80f5488d08971fdc2d"
			}
		  }`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/borales/actions-yarn/git/ref/tags/v2.3.0",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v2.3.0",
			"node_id": "MDM6UmVmMTcyMjA5NzEzOnJlZnMvaGVhZHMvbWFzdGVy",
			"url": "https://api.github.com/repos/borales/actions-yarn/git/ref/tags/v2.3.0",
			"object": {
			  "sha": "4965e1a0f0ae9c422a9a5748ebd1fb5e097d22b9",
			  "type": "commit",
			  "url": "https://api.github.com/repos/borales/actions-yarn/git/commits/4965e1a0f0ae9c422a9a5748ebd1fb5e097d22b9"
			}
		  }`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/git/ref/tags/v1",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v1",
			"node_id": "MDM6UmVmMTk3ODE0NjI5OnJlZnMvdGFncy92MQ==",
			"url": "https://api.github.com/repos/actions/checkout/git/refs/tags/v1",
			"object": {
			  "sha": "544eadc6bf3d226fd7a7a9f0dc5b5bf7ca0675b9",
			  "type": "tag",
			  "url": "https://api.github.com/repos/actions/checkout/git/tags/544eadc6bf3d226fd7a7a9f0dc5b5bf7ca0675b9"
			}
		  }`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/setup-node/git/ref/tags/v1",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v1",
			"node_id": "MDM6UmVmMTg5NDc2OTA0OnJlZnMvdGFncy92MQ==",
			"url": "https://api.github.com/repos/actions/setup-node/git/refs/tags/v1",
			"object": {
			  "sha": "f1f314fca9dfce2769ece7d933488f076716723e",
			  "type": "commit",
			  "url": "https://api.github.com/repos/actions/setup-node/git/commits/f1f314fca9dfce2769ece7d933488f076716723e"
			}
		  }`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/JS-DevTools/npm-publish/git/ref/tags/v1",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v1",
			"node_id": "MDM6UmVmMjM1MjMxNTEzOnJlZnMvdGFncy92MQ==",
			"url": "https://api.github.com/repos/JS-DevTools/npm-publish/git/refs/tags/v1",
			"object": {
			  "sha": "22595ff8c4d0d9f53cef0656fbb90fbe06ee885c",
			  "type": "tag",
			  "url": "https://api.github.com/repos/JS-DevTools/npm-publish/git/tags/22595ff8c4d0d9f53cef0656fbb90fbe06ee885c"
			}
		  }`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/brandedoutcast/publish-nuget/git/ref/tags/v2",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v2",
			"node_id": "MDM6UmVmMjI4MTk2ODk5OnJlZnMvdGFncy92Mg==",
			"url": "https://api.github.com/repos/brandedoutcast/publish-nuget/git/refs/tags/v2",
			"object": {
			  "sha": "c12b8546b67672ee38ac87bea491ac94a587f7cc",
			  "type": "commit",
			  "url": "https://api.github.com/repos/brandedoutcast/publish-nuget/git/commits/c12b8546b67672ee38ac87bea491ac94a587f7cc"
			}
		  }`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/rohith/publish-nuget/git/ref/tags/v2",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v2",
			"node_id": "MDM6UmVmMjI4MTk2ODk5OnJlZnMvdGFncy92Mg==",
			"url": "https://api.github.com/repos/brandedoutcast/publish-nuget/git/refs/tags/v2",
			"object": {
			  "sha": "c12b8546b67672ee38ac87bea491ac94a587f7cc",
			  "type": "commit",
			  "url": "https://api.github.com/repos/brandedoutcast/publish-nuget/git/commits/c12b8546b67672ee38ac87bea491ac94a587f7cc"
			}
		  }`))
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
