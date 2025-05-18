package workflow

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/step-security/secure-repo/remediation/workflow/maintainedactions"
	"github.com/step-security/secure-repo/remediation/workflow/permissions"
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

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/action-semantic-pull-request/releases/latest",
		httpmock.NewStringResponder(200, `{
		"tag_name": "v5.5.5",
		"name": "v5.5.5",
		"body": "Release notes",
		"created_at": "2023-01-01T00:00:00Z"
	}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/git-restore-mtime-action/releases/latest",
		httpmock.NewStringResponder(200, `{
			"tag_name": "v2.1.0",
			"name": "v2.1.0",
			"body": "Release notes",
			"created_at": "2023-01-01T00:00:00Z"
		}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/github/super-linter/releases/latest",
		httpmock.NewStringResponder(200, `{
			"tag_name": "v4.9.0",
			"name": "v4.9.0",
			"body": "Release notes",
			"created_at": "2023-01-01T00:00:00Z"
		}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/skip-duplicate-actions/releases/latest",
		httpmock.NewStringResponder(200, `{
			"tag_name": "v2.1.0",
			"name": "v2.1.0",
			"body": "Release notes",
			"created_at": "2023-01-01T00:00:00Z"
		}`))

	// Mock APIs for step-security/action-semantic-pull-request
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/action-semantic-pull-request/commits/v5",
		httpmock.NewStringResponder(200, `a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/action-semantic-pull-request/git/matching-refs/tags/v5.",
		httpmock.NewStringResponder(200, `[
			{
				"ref": "refs/tags/v5.5.5",
				"object": {
					"sha": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0",
					"type": "commit"
				}
			}
		]`))

	// Mock APIs for step-security/skip-duplicate-actions
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/skip-duplicate-actions/commits/v2",
		httpmock.NewStringResponder(200, `b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0a1`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/skip-duplicate-actions/git/matching-refs/tags/v2.",
		httpmock.NewStringResponder(200, `[
			{
				"ref": "refs/tags/v2.1.0",
				"object": {
					"sha": "b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0a1",
					"type": "commit"
				}
			}
		]`))

	// Mock APIs for step-security/git-restore-mtime-action
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/git-restore-mtime-action/commits/v2",
		httpmock.NewStringResponder(200, `c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0a1b2`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/git-restore-mtime-action/git/matching-refs/tags/v2.",
		httpmock.NewStringResponder(200, `[
			{
				"ref": "refs/tags/v2.1.0",
				"object": {
					"sha": "c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0a1b2",
					"type": "commit"
				}
			}
		]`))

	tests := []struct {
		fileName                   string
		wantPinnedActions          bool
		wantAddedHardenRunner      bool
		wantAddedPermissions       bool
		wantAddedMaintainedActions bool
	}{
		{fileName: "oneJob.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: false, wantAddedMaintainedActions: true},
		{fileName: "allscenarios.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: true},
		{fileName: "missingaction.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: false},
		{fileName: "nohardenrunner.yml", wantPinnedActions: true, wantAddedHardenRunner: false, wantAddedPermissions: true},
		{fileName: "noperms.yml", wantPinnedActions: true, wantAddedHardenRunner: true, wantAddedPermissions: false},
		{fileName: "nopin.yml", wantPinnedActions: false, wantAddedHardenRunner: true, wantAddedPermissions: true},
		{fileName: "allperms.yml", wantPinnedActions: false, wantAddedHardenRunner: false, wantAddedPermissions: true},
		{fileName: "multiplejobperms.yml", wantPinnedActions: false, wantAddedHardenRunner: false, wantAddedPermissions: true},
		{fileName: "error.yml", wantPinnedActions: false, wantAddedHardenRunner: false, wantAddedPermissions: false},
	}
	for _, test := range tests {
		var err error
		var input []byte
		input, err = ioutil.ReadFile(path.Join(inputDirectory, test.fileName))

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
		case "oneJob.yml":
			queryParams["addMaintainedActions"] = "true"
			queryParams["addHardenRunner"] = "true"
			queryParams["pinActions"] = "true"
			queryParams["addPermissions"] = "false"
		}
		queryParams["addProjectComment"] = "false"

		var output *permissions.SecureWorkflowReponse
		var actionMap map[string]string
		if test.fileName == "oneJob.yml" {
			actionMap, err = maintainedactions.LoadMaintainedActions("maintainedactions/maintainedActions.json")
			if err != nil {
				t.Errorf("unable to load the file %s", err)
			}
			output, err = SecureWorkflow(queryParams, string(input), &mockDynamoDBClient{}, []string{}, false, actionMap)

		} else {
			output, err = SecureWorkflow(queryParams, string(input), &mockDynamoDBClient{})
		}

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

func TestSecureWorkflowContainerJob(t *testing.T) {
	const inputDirectory = "../../testfiles/secureworkflow/input"
	const outputDirectory = "../../testfiles/secureworkflow/output"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock APIs for actions/checkout
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/commits/v3",
		httpmock.NewStringResponder(200, `c85c95e3d7251135ab7dc9ce3241c5835cc595a9`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/git/matching-refs/tags/v3.",
		httpmock.NewStringResponder(200,
			`[
				{
				  "ref": "refs/tags/v3.5.3",
				  "object": {
					"sha": "c85c95e3d7251135ab7dc9ce3241c5835cc595a9",
					"type": "commit"
				  }
				}
			  ]`),
	)

	// Mock APIs for step-security/harden-runner
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/commits/v2",
		httpmock.NewStringResponder(200, `17d0e2bd7d51742c71671bd19fa12bdc9d40a3d6`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/git/matching-refs/tags/v2.",
		httpmock.NewStringResponder(200,
			`[
				{
				  "ref": "refs/tags/v2.8.1",
				  "object": {
					"sha": "17d0e2bd7d51742c71671bd19fa12bdc9d40a3d6",
					"type": "commit"
				  }
				}
			  ]`),
	)

	var err error
	var input []byte
	input, err = ioutil.ReadFile(path.Join(inputDirectory, "container-job.yml"))

	if err != nil {
		log.Fatal(err)
	}

	os.Setenv("KBFolder", "../../knowledge-base/actions")

	// Test with skipHardenRunnerForContainers = true
	queryParams := make(map[string]string)
	queryParams["skipHardenRunnerForContainers"] = "true"
	queryParams["addProjectComment"] = "false"

	output, err := SecureWorkflow(queryParams, string(input), &mockDynamoDBClient{})

	if err != nil {
		t.Errorf("Error not expected")
	}

	// Verify that harden runner was not added
	if output.AddedHardenRunner {
		t.Errorf("Harden runner should not be added for container job with skipHardenRunnerForContainers=true")
	}

	// Verify that the output matches expected output file
	expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, "container-job.yml"))
	if err != nil {
		log.Fatal(err)
	}

	if output.FinalOutput != string(expectedOutput) {
		t.Errorf("test failed container-job.yml did not match expected output\nExpected:\n%s\n\nGot:\n%s",
			string(expectedOutput), output.FinalOutput)
	}

	// Verify permissions were added
	if !output.AddedPermissions {
		t.Errorf("Permissions should be added even for container jobs")
	}

	// Verify actions were pinned
	if !output.PinnedActions {
		t.Errorf("Actions should be pinned even for container jobs")
	}
}

func TestSecureWorkflowEmptyPermissions(t *testing.T) {
	const inputDirectory = "../../testfiles/secureworkflow/input"
	const outputDirectory = "../../testfiles/secureworkflow/output"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock APIs for actions/checkout
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/commits/v2",
		httpmock.NewStringResponder(200, `ee0669bd1cc54295c223e0bb666b733df41de1c5`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/actions/checkout/git/matching-refs/tags/v2.",
		httpmock.NewStringResponder(200,
			`[
				{
				  "ref": "refs/tags/v2.7.0",
				  "object": {
					"sha": "ee0669bd1cc54295c223e0bb666b733df41de1c5",
					"type": "commit"
				  }
				}
			  ]`),
	)

	// Mock APIs for actions/setup-node
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
			  ]`),
	)

	// Mock APIs for step-security/harden-runner
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/commits/v2",
		httpmock.NewStringResponder(200, `17d0e2bd7d51742c71671bd19fa12bdc9d40a3d6`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/git/matching-refs/tags/v2.",
		httpmock.NewStringResponder(200,
			`[
				{
				  "ref": "refs/tags/v2.8.1",
				  "object": {
					"sha": "17d0e2bd7d51742c71671bd19fa12bdc9d40a3d6",
					"type": "commit"
				  }
				}
			  ]`),
	)

	var err error
	var input []byte
	input, err = ioutil.ReadFile(path.Join(inputDirectory, "empty-permissions.yml"))

	if err != nil {
		log.Fatal(err)
	}

	os.Setenv("KBFolder", "../../knowledge-base/actions")

	queryParams := make(map[string]string)
	queryParams["addEmptyTopLevelPermissions"] = "true"
	queryParams["addProjectComment"] = "false"

	output, err := SecureWorkflow(queryParams, string(input), &mockDynamoDBClient{})

	if err != nil {
		t.Errorf("Error not expected")
	}

	expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, "empty-permissions.yml"))

	if err != nil {
		log.Fatal(err)
	}

	if output.FinalOutput != string(expectedOutput) {
		// Write the actual output to a file for debugging
		debugFile := path.Join(outputDirectory, "empty-permissions-debug.yml")
		err := ioutil.WriteFile(debugFile, []byte(output.FinalOutput), 0644)
		if err != nil {
			t.Logf("Failed to write debug file: %v", err)
		} else {
			t.Logf("Actual output written to: %s", debugFile)
		}

		t.Errorf("test failed empty-permissions.yml did not match expected output\nExpected:\n%s\n\nGot:\n%s",
			string(expectedOutput), output.FinalOutput)
	}

}
