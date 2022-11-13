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
		input, err := ioutil.ReadFile(path.Join(inputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		os.Setenv("KBFolder", "../../../knowledge-base/actions")

		fixWorkflowPermsResponse, err := AddJobLevelPermissions(string(input))
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

		input, err := ioutil.ReadFile(path.Join(inputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		addProjectComment := false

		switch f.Name() {
		case "addprojectcomment.yml":
			addProjectComment = true
		}

		output, err := AddWorkflowLevelPermissions(string(input), addProjectComment)

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
