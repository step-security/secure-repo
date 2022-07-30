package main

import (
	"io/ioutil"
	"log"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestSecureWorkflow(t *testing.T) {
	const inputDirectory = "./testfiles/secureworkflow/input"
	const outputDirectory = "./testfiles/secureworkflow/output"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

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

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/git/ref/tags/v1",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v1",
			"node_id": "REF_kwDOGSuXyq9yZWZzL2hlYWRzL21haW4",
			"url": "https://api.github.com/repos/step-security/harden-runner/git/refs/tags/v1",
			"object": {
				"sha": "7206db2ec98c5538323a6d70e51f965d55c11c87",
				"type": "commit",
				"url": "https://api.github.com/repos/step-security/harden-runner/git/commits/7206db2ec98c5538323a6d70e51f965d55c11c87"
			}
		}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/github/super-linter/git/ref/tags/v3",
		httpmock.NewStringResponder(200, `{
			"ref": "refs/tags/v3",
			"node_id": "MDM6UmVmMjE2NTgxNTY3OnJlZnMvdGFncy92Mw==",
			"url": "https://api.github.com/repos/github/super-linter/git/refs/tags/v3",
			"object": {
				"sha": "34b2f8032d759425f6b42ea2e52231b33ae05401",
				"type": "commit",
				"url": "https://api.github.com/repos/github/super-linter/git/commits/34b2f8032d759425f6b42ea2e52231b33ae05401"
			}
		}`))

	tests := []struct {
		fileName              string
		wantPinnedActions     bool
		wantAddedHardenRunner bool
		wantAddedPermissions  bool
	}{
		{fileName: "allscenarios.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: true},
		{fileName: "missingaction.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: true},
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
