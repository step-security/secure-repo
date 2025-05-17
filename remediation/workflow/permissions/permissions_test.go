package permissions

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"testing"
)

func TestAddJobLevelPermissions(t *testing.T) {
	const inputDirectory = "../../../testfiles/joblevelpermskb/input"
	const outputDirectory = "../../../testfiles/joblevelpermskb/output"
	files, err := ioutil.ReadDir(inputDirectory)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {

		if f.Name() == "empty-top-level-permissions.yml" {
			continue
		}

		input, err := ioutil.ReadFile(path.Join(inputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		os.Setenv("KBFolder", "../../../knowledge-base/actions")

		fixWorkflowPermsResponse, err := AddJobLevelPermissions(string(input), false)
		output := fixWorkflowPermsResponse.FinalOutput
		jobErrors := fixWorkflowPermsResponse.JobErrors

		// some test cases return a job error for known issues
		if len(jobErrors) > 0 {
			for _, je := range jobErrors {
				if strings.Compare(je.JobName, "job-with-error") == 0 {
					if strings.Contains(je.Errors[0], "KnownIssue") {
						output = je.Errors[0]
					} else {
						t.Errorf("test failed. unexpected job error %s, error: %v", f.Name(), jobErrors)
					}
				}
			}

		}

		if fixWorkflowPermsResponse.AlreadyHasPermissions {
			output = errorAlreadyHasPermissions
		}

		if fixWorkflowPermsResponse.IncorrectYaml {
			output = errorIncorrectYaml
		}

		if err != nil {
			t.Errorf("test failed %s, error: %v", f.Name(), err)
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		if output != string(expectedOutput) {
			t.Errorf("test failed %s did not match expected output\n%s", f.Name(), output)
		}
	}
}

func TestAddJobLevelPermissionsWithEmptyTopLevel(t *testing.T) {
	const inputDirectory = "../../../testfiles/joblevelpermskb/input"
	const outputDirectory = "../../../testfiles/joblevelpermskb/output"

	// Test the empty-top-level-permissions.yml file
	input, err := ioutil.ReadFile(path.Join(inputDirectory, "empty-top-level-permissions.yml"))
	if err != nil {
		t.Fatal(err)
	}

	expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, "empty-top-level-permissions.yml"))
	if err != nil {
		t.Fatal(err)
	}

	os.Setenv("KBFolder", "../../../knowledge-base/actions")

	// Test with addEmptyTopLevelPermissions = true
	fixWorkflowPermsResponse, err := AddJobLevelPermissions(string(input), true)
	if err != nil {
		t.Errorf("Unexpected error with addEmptyTopLevelPermissions=true: %v", err)
	}

	if fixWorkflowPermsResponse.FinalOutput != string(expectedOutput) {
		t.Errorf("test failed with addEmptyTopLevelPermissions=true for empty-top-level-permissions.yml\nExpected:\n%s\n\nGot:\n%s",
			string(expectedOutput), fixWorkflowPermsResponse.FinalOutput)
	}

	// Test with addEmptyTopLevelPermissions = false (should skip contents: read)
	fixWorkflowPermsResponse2, err2 := AddJobLevelPermissions(string(input), false)
	if err2 != nil {
		t.Errorf("Unexpected error with addEmptyTopLevelPermissions=false: %v", err2)
	}

	// With false, contents: read should be skipped at job level
	if fixWorkflowPermsResponse2.FinalOutput != string(input) {
		t.Errorf("test failed with addEmptyTopLevelPermissions=false for empty-top-level-permissions.yml\nExpected:\n%s\n\nGot:\n%s",
			string(input), fixWorkflowPermsResponse2.FinalOutput)
	}
}

func Test_addPermissions(t *testing.T) {
	type args struct {
		inputYaml   string
		jobName     string
		permissions []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "bad yaml",
			args: args{
				inputYaml: "123",
			}, want: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addPermissions(tt.args.inputYaml, tt.args.jobName, tt.args.permissions)
			if (err != nil) != tt.wantErr {
				t.Errorf("addPermissions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("addPermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddWorkflowLevelPermissions(t *testing.T) {
	const inputDirectory = "../../../testfiles/toplevelperms/input"
	const outputDirectory = "../../../testfiles/toplevelperms/output"
	files, err := ioutil.ReadDir(inputDirectory)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".yml") {
			continue
		}

		if f.Name() == "empty-permissions.yml" {
			continue
		}

		input, err := ioutil.ReadFile(path.Join(inputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		addProjectComment := false

		switch f.Name() {
		case "addprojectcomment.yml":
			addProjectComment = true
		}

		output, err := AddWorkflowLevelPermissions(string(input), addProjectComment, false)

		if err != nil {
			t.Errorf("Error not expected")
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		if output != string(expectedOutput) {
			t.Errorf("test failed %s did not match expected output\n%s", f.Name(), output)
		}
	}

}

func TestAddWorkflowLevelPermissionsWithEmpty(t *testing.T) {
	const inputDirectory = "../../../testfiles/toplevelperms/input"
	const outputDirectory = "../../../testfiles/toplevelperms/output"

	// Test the empty-permissions.yml file
	input, err := ioutil.ReadFile(path.Join(inputDirectory, "empty-permissions.yml"))
	if err != nil {
		t.Fatal(err)
	}

	expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, "empty-permissions.yml"))
	if err != nil {
		t.Fatal(err)
	}

	// Test with addEmptyTopLevelPermissions = true
	output, err := AddWorkflowLevelPermissions(string(input), false, true)
	if err != nil {
		t.Errorf("Unexpected error with addEmptyTopLevelPermissions=true: %v", err)
	}

	if output != string(expectedOutput) {
		t.Errorf("test failed with addEmptyTopLevelPermissions=true for empty-permissions.yml\nExpected:\n%s\n\nGot:\n%s",
			string(expectedOutput), output)
	}

	// Test with addEmptyTopLevelPermissions = false (should add contents: read)
	output2, err2 := AddWorkflowLevelPermissions(string(input), false, false)
	if err2 != nil {
		t.Errorf("Unexpected error with addEmptyTopLevelPermissions=false: %v", err2)
	}

	// With false, should add contents: read instead of empty permissions
	if !strings.Contains(output2, "contents: read") || strings.Contains(output2, "permissions: {}") {
		t.Errorf("test failed with addEmptyTopLevelPermissions=false for empty-permissions.yml - should contain 'contents: read' but not 'permissions: {}'\nGot:\n%s", output2)
	}
}
