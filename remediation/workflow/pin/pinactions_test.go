package pin

import (
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestPinActions(t *testing.T) {
	const inputDirectory = "../../../testfiles/pinactions/input"
	const outputDirectory = "../../../testfiles/pinactions/output"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/peter-evans/close-issue/commits/v1",
		httpmock.NewStringResponder(200, `a700eac5bf2a1c7a8cb6da0c13f93ed96fd53dbe`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/peter-evans/close-issue/git/matching-refs/tags/v1.",
		httpmock.NewStringResponder(200,
			`[
				{
					"ref": "refs/tags/v1.0.3",
					"object": {
					"sha": "a700eac5bf2a1c7a8cb6da0c13f93ed96fd53dbe",
					"type": "commit"
					}
				}
			]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/commits/master",
		httpmock.NewStringResponder(200, `61b9e3751b92087fd0b06925ba6dd6314e06f089`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/git/matching-refs/tags/master.",
		httpmock.NewStringResponder(200, `[]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/elgohr/Publish-Docker-Github-Action/commits/master",
		httpmock.NewStringResponder(200, `8217e91c0369a5342a4ef2d612de87492410a666`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/elgohr/Publish-Docker-Github-Action/git/matching-refs/tags/master.",
		httpmock.NewStringResponder(200, `[]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/borales/actions-yarn/commits/v2.3.0",
		httpmock.NewStringResponder(200, `4965e1a0f0ae9c422a9a5748ebd1fb5e097d22b9`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/borales/actions-yarn/git/matching-refs/tags/v2.3.0.",
		httpmock.NewStringResponder(200, `[]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/commits/v1",
		httpmock.NewStringResponder(200, `544eadc6bf3d226fd7a7a9f0dc5b5bf7ca0675b9`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/git/matching-refs/tags/v1.",
		httpmock.NewStringResponder(200,
			`[
				{
				  "ref": "refs/tags/v1.0.0",
				  "node_id": "MDM6UmVmMTk3ODE0NjI5OnJlZnMvdGFncy92MS4wLjA=",
				  "url": "https://api.github.com/repos/actions/checkout/git/refs/tags/v1.0.0",
				  "object": {
					"sha": "af513c7a016048ae468971c52ed77d9562c7c819",
					"type": "commit",
					"url": "https://api.github.com/repos/actions/checkout/git/commits/af513c7a016048ae468971c52ed77d9562c7c819"
				  }
				},
				{
				  "ref": "refs/tags/v1.1.0",
				  "node_id": "MDM6UmVmMTk3ODE0NjI5OnJlZnMvdGFncy92MS4xLjA=",
				  "url": "https://api.github.com/repos/actions/checkout/git/refs/tags/v1.1.0",
				  "object": {
					"sha": "ec3afacf7f605c9fc12c70bc1c9e1708ddb99eca",
					"type": "tag",
					"url": "https://api.github.com/repos/actions/checkout/git/tags/ec3afacf7f605c9fc12c70bc1c9e1708ddb99eca"
				  }
				},
				{
				  "ref": "refs/tags/v1.2.0",
				  "node_id": "MDM6UmVmMTk3ODE0NjI5OnJlZnMvdGFncy92MS4yLjA=",
				  "url": "https://api.github.com/repos/actions/checkout/git/refs/tags/v1.2.0",
				  "object": {
					"sha": "a2ca40438991a1ab62db1b7cad0fd4e36a2ac254",
					"type": "tag",
					"url": "https://api.github.com/repos/actions/checkout/git/tags/a2ca40438991a1ab62db1b7cad0fd4e36a2ac254"
				  }
				}
			  ]`),
	)

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/commits/v1.2.0",
		httpmock.NewStringResponder(200, `544eadc6bf3d226fd7a7a9f0dc5b5bf7ca0675b9`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/setup-node/commits/v1",
		httpmock.NewStringResponder(200, `f1f314fca9dfce2769ece7d933488f076716723e`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/setup-node/git/matching-refs/tags/v1.",
		httpmock.NewStringResponder(200,
			`[
				{
					"ref": "refs/tags/v1.4.6",
					"object": {
					  "sha": "f1f314fca9dfce2769ece7d933488f076716723e",
					  "type": "commit"
					}
				  }
			]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/JS-DevTools/npm-publish/commits/v1",
		httpmock.NewStringResponder(200, `0f451a94170d1699fd50710966d48fb26194d939`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/JS-DevTools/npm-publish/git/matching-refs/tags/v1.",
		httpmock.NewStringResponder(200,
			`[
				{
					"ref": "refs/tags/v1.4.3",
					"object": {
					  "sha": "0f451a94170d1699fd50710966d48fb26194d939",
					  "type": "commit"
					}
				  }
			]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/brandedoutcast/publish-nuget/commits/v2",
		httpmock.NewStringResponder(200, `c12b8546b67672ee38ac87bea491ac94a587f7cc`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/brandedoutcast/publish-nuget/git/matching-refs/tags/v2.",
		httpmock.NewStringResponder(200,
			`[
				{
					"ref": "refs/tags/v2.5.3",
					"node_id": "MDM6UmVmMjI4MTk2ODk5OnJlZnMvdGFncy92Mi41LjM=",
					"url": "https://api.github.com/repos/brandedoutcast/publish-nuget/git/refs/tags/v2.5.3",
					"object": {
					  "sha": "4637c3bdd3fb4c052235299664c57b14c398cbd0",
					  "type": "commit",
					  "url": "https://api.github.com/repos/brandedoutcast/publish-nuget/git/commits/4637c3bdd3fb4c052235299664c57b14c398cbd0"
					}
				},
				{
					"ref": "refs/tags/v2.5.4",
					"node_id": "MDM6UmVmMjI4MTk2ODk5OnJlZnMvdGFncy92Mi41LjQ=",
					"url": "https://api.github.com/repos/brandedoutcast/publish-nuget/git/refs/tags/v2.5.4",
					"object": {
					  "sha": "108c10b32aa03efa5f71af6a233dc2e8e32845cb",
					  "type": "commit",
					  "url": "https://api.github.com/repos/brandedoutcast/publish-nuget/git/commits/108c10b32aa03efa5f71af6a233dc2e8e32845cb"
					}
				},
				{
					"ref": "refs/tags/v2.5.5",
					"object": {
					  "sha": "c12b8546b67672ee38ac87bea491ac94a587f7cc",
					  "type": "commit"
					}
				}
			]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/rohith/publish-nuget/commits/v2",
		httpmock.NewStringResponder(200, `c12b8546b67672ee38ac87bea491ac94a587f7cc`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/rohith/publish-nuget/git/matching-refs/tags/v2.",
		httpmock.NewStringResponder(200,
			`[
				{
					"ref": "refs/tags/v2.5.5",
					"object": {
					  "sha": "c12b8546b67672ee38ac87bea491ac94a587f7cc",
					  "type": "commit"
					}
				  }
			]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/github/codeql-action/commits/v3",
		httpmock.NewStringResponder(200, `d68b2d4edb4189fd2a5366ac14e72027bd4b37dd`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/github/codeql-action/git/matching-refs/tags/v3.",
		httpmock.NewStringResponder(200,
			`[
				{
					"ref": "refs/tags/v3.28.2",
					"object": {
					  "sha": "d68b2d4edb4189fd2a5366ac14e72027bd4b37dd",
					  "type": "commit"
					}
				  }
			]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/github/codeql-action/commits/v3.28.2",
		httpmock.NewStringResponder(200, `d68b2d4edb4189fd2a5366ac14e72027bd4b37dd`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/github/codeql-action/git/matching-refs/tags/v3.28.2.",
		httpmock.NewStringResponder(200,
			`[
					{
						"ref": "refs/tags/v3.28.2",
						"object": {
						  "sha": "d68b2d4edb4189fd2a5366ac14e72027bd4b37dd",
						  "type": "commit"
						}
					  }
				]`))

	// mock ping response
	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/",
		httpmock.NewStringResponder(200, ``))

	// Mock token endpoints
	httpmock.RegisterResponder("GET", "https://ghcr.io/token",
		func(req *http.Request) (*http.Response, error) {
			scope := req.URL.Query().Get("scope")
			switch scope {
			// Following are the ones which simulate the image existance in ghcr
			case "repository:actions/checkout:pull",
				"repository:step-security/wait-for-secrets:pull",
				"repository:actions/setup-node:pull",
				"repository:peter-evans/close-issue:pull",
				"repository:borales/actions-yarn:pull",
				"repository:JS-DevTools/npm-publish:pull",
				"repository:elgohr/Publish-Docker-Github-Action:pull",
				"repository:brandedoutcast/publish-nuget:pull",
				"repository:rohith/publish-nuget:pull",
				"repository:github/codeql-action:pull":
				return httpmock.NewJsonResponse(http.StatusOK, map[string]string{
					"token":        "test-token",
					"access_token": "test-token",
				})
			default:
				return httpmock.NewJsonResponse(http.StatusForbidden, map[string]interface{}{
					"errors": []map[string]string{
						{
							"code":    "DENIED",
							"message": "requested access to the resource is denied",
						},
					},
				})
			}
		})

	// Mock manifest endpoints for specific versions and commit hashes
	manifestResponders := []string{
		// the following list will contain the list of actions with versions
		// which are mocked to be immutable
		"actions/checkout@v1.2.0",
		"github/codeql-action@v3.28.2",
	}

	for _, action := range manifestResponders {
		actionPath := strings.Split(action, "@")[0]
		version := strings.TrimPrefix(strings.Split(action, "@")[1], "v")
		// Mock manifest response so that we can treat action as immutable
		httpmock.RegisterResponder("GET", "https://ghcr.io/v2/"+actionPath+"/manifests/"+version,
			func(req *http.Request) (*http.Response, error) {
				return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
					"schemaVersion": 2,
					"mediaType":     "application/vnd.github.actions.package.v1+json",
					"artifactType":  "application/vnd.github.actions.package.v1+json",
					"config": map[string]interface{}{
						"mediaType": "application/vnd.github.actions.package.v1+json",
					},
				})
			})
	}

	// Default manifest response for non-existent tags
	httpmock.RegisterResponder("GET", `=~^https://ghcr\.io/v2/.*/manifests/.*`,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusNotFound, map[string]interface{}{
				"errors": []map[string]string{
					{
						"code":    "MANIFEST_UNKNOWN",
						"message": "manifest unknown",
					},
				},
			})
		})

	tests := []struct {
		fileName        string
		wantUpdated     bool
		exemptedActions []string
		pinToImmutable  bool
	}{
		{fileName: "alreadypinned.yml", wantUpdated: false, pinToImmutable: true},
		{fileName: "branch.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "localaction.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "multiplejobs.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "basic.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "dockeraction.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "multipleactions.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "actionwithcomment.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "repeatedactionwithcomment.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "immutableaction-1.yml", wantUpdated: true, pinToImmutable: true},
		{fileName: "exemptaction.yml", wantUpdated: true, exemptedActions: []string{"actions/checkout", "rohith/*"}, pinToImmutable: true},
		{fileName: "donotpintoimmutable.yml", wantUpdated: true, pinToImmutable: false},
	}
	for _, tt := range tests {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.fileName))

		if err != nil {
			log.Fatal(err)
		}

		output, gotUpdated, err := PinActions(string(input), tt.exemptedActions, tt.pinToImmutable)
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
