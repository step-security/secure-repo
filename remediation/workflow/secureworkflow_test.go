package workflow

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestSecureWorkflow(t *testing.T) {
	const inputDirectory = "../../testfiles/secureworkflow/input"
	const outputDirectory = "../../testfiles/secureworkflow/output"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

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

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/commits/v2",
		httpmock.NewStringResponder(200, `ebacdc22ef6c2cfb85ee5ded8f2e640f4c776dd5`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/git/matching-refs/tags/v2.",
		httpmock.NewStringResponder(200,
			`[
				{
				  "ref": "refs/tags/v2.0.0",
				  "node_id": "REF_kwDOGSuXyrByZWZzL3RhZ3MvdjIuMC4w",
				  "url": "https://api.github.com/repos/step-security/harden-runner/git/refs/tags/v2.0.0",
				  "object": {
					"sha": "ebacdc22ef6c2cfb85ee5ded8f2e640f4c776dd5",
					"type": "commit",
					"url": "https://api.github.com/repos/step-security/harden-runner/git/commits/ebacdc22ef6c2cfb85ee5ded8f2e640f4c776dd5"
				  }
				}
			  ]`),
	)

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/github/super-linter/commits/v3",
		httpmock.NewStringResponder(200, `34b2f8032d759425f6b42ea2e52231b33ae05401`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/github/super-linter/git/matching-refs/tags/v3.",
		httpmock.NewStringResponder(200,
			`[
				{
					"ref": "refs/tags/v3.17.0",
					"node_id": "MDM6UmVmMjE2NTgxNTY3OnJlZnMvdGFncy92My4xNy4w",
					"url": "https://api.github.com/repos/github/super-linter/git/refs/tags/v3.17.0",
					"object": {
					  "sha": "28cfebb84fd6dd9e8773b5efe5ac0f8f3714f228",
					  "type": "commit",
					  "url": "https://api.github.com/repos/github/super-linter/git/commits/28cfebb84fd6dd9e8773b5efe5ac0f8f3714f228"
					}
				  },
				  {
					"ref": "refs/tags/v3.17.1",
					"node_id": "MDM6UmVmMjE2NTgxNTY3OnJlZnMvdGFncy92My4xNy4x",
					"url": "https://api.github.com/repos/github/super-linter/git/refs/tags/v3.17.1",
					"object": {
					  "sha": "34b2f8032d759425f6b42ea2e52231b33ae05401",
					  "type": "commit",
					  "url": "https://api.github.com/repos/github/super-linter/git/commits/34b2f8032d759425f6b42ea2e52231b33ae05401"
					}
				  }
			  ]`),
	)

	tests := []struct {
		fileName              string
		wantPinnedActions     bool
		wantAddedHardenRunner bool
		wantAddedPermissions  bool
	}{
		{fileName: "allscenarios.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: true},
		{fileName: "missingaction.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: false},
		{fileName: "nohardenrunner.yml", wantPinnedActions: true, wantAddedHardenRunner: false, wantAddedPermissions: true},
		{fileName: "noperms.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: false},
		{fileName: "nopin.yml", wantPinnedActions: false, wantAddedHardenRunner: true, wantAddedPermissions: true},
		{fileName: "allperms.yml", wantPinnedActions: false, wantAddedHardenRunner: false, wantAddedPermissions: true},
		{fileName: "multiplejobperms.yml", wantPinnedActions: false, wantAddedHardenRunner: false, wantAddedPermissions: true},
	}
	for _, test := range tests {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.fileName))

		if err != nil {
			log.Fatal(err)
		}

		os.Setenv("KBFolder", "../../knowledge-base/actions")

		queryParams := make(map[string]string)
		switch test.fileName {
		case "nopin.yml":
			queryParams["pinActions"] = "false"
		case "nohardenrunner.yml":
			queryParams["addHardenRunner"] = "false"
		case "noperms.yml":
			queryParams["addPermissions"] = "false"
		case "allperms.yml":
			queryParams["addHardenRunner"] = "false"
			queryParams["pinActions"] = "false"
		case "multiplejobperms.yml":
			queryParams["addHardenRunner"] = "false"
			queryParams["pinActions"] = "false"
		}
		queryParams["addProjectComment"] = "false"

		output, err := SecureWorkflow(queryParams, string(input), &mockDynamoDBClient{})

		if err != nil {
			t.Errorf("Error not expected")
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, test.fileName))

		if err != nil {
			log.Fatal(err)
		}

		if output.FinalOutput != string(expectedOutput) {
			t.Errorf("test failed %s did not match expected output\n%s", test.fileName, output.FinalOutput)
		}

		if output.AddedHardenRunner != test.wantAddedHardenRunner {
			t.Errorf("test failed %s did not match expected AddedHardenRunner value. Expected:%v Actual:%v", test.fileName, test.wantAddedHardenRunner, output.AddedHardenRunner)
		}

		if output.AddedPermissions != test.wantAddedPermissions {
			t.Errorf("test failed %s did not match expected AddedPermissions value. Expected:%v Actual:%v", test.fileName, test.wantAddedPermissions, output.AddedPermissions)
		}

		if output.PinnedActions != test.wantPinnedActions {
			t.Errorf("test failed %s did not match expected PinnedActions value. Expected:%v Actual:%v", test.fileName, test.wantPinnedActions, output.PinnedActions)
		}
	}

}
