package hardenrunner

import (
	"io/ioutil"
	"path"
	"testing"
)

func TestAddAction(t *testing.T) {
	type args struct {
		inputYaml string
		action    string
	}
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"
	tests := []struct {
		name        string
		args        args
		want        string
		wantErr     bool
		wantUpdated bool
	}{
		{name: "one job", args: args{inputYaml: "action-issues.yml", action: "step-security/harden-runner@v2"}, want: "action-issues.yml", wantErr: false, wantUpdated: true},
		{name: "two jobs", args: args{inputYaml: "2jobs.yml", action: "step-security/harden-runner@v2"}, want: "2jobs.yml", wantErr: false, wantUpdated: true},
		{name: "already present", args: args{inputYaml: "alreadypresent.yml", action: "step-security/harden-runner@v2"}, want: "alreadypresent.yml", wantErr: false, wantUpdated: true},
		{name: "already present 2", args: args{inputYaml: "alreadypresent_2.yml", action: "step-security/harden-runner@v2"}, want: "alreadypresent_2.yml", wantErr: false, wantUpdated: false},
		{name: "reusable job", args: args{inputYaml: "reusablejob.yml", action: "step-security/harden-runner@v2"}, want: "reusablejob.yml", wantErr: false, wantUpdated: false},
		{name: "job name in input", args: args{inputYaml: "jobNameInInput.yml", action: "step-security/harden-runner@v2"}, want: "jobNameInInput.yml", wantErr: false, wantUpdated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.args.inputYaml))
			if err != nil {
				t.Fatalf("error reading test file")
			}
			got, gotUpdated, err := AddAction(string(input), tt.args.action, false, false, false)

			if gotUpdated != tt.wantUpdated {
				t.Errorf("AddAction() updated = %v, wantUpdated %v", gotUpdated, tt.wantUpdated)
				return
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("AddAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			output, err := ioutil.ReadFile(path.Join(outputDirectory, tt.args.inputYaml))
			if err != nil {
				t.Fatalf("error reading test file")
			}
			if got != string(output) {
				t.Errorf("AddAction() = %v, want %v", got, string(output))
			}
		})
	}
}

func TestAddActionWithContainer(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"
	
	// Test container job with skipContainerJobs = true
	input, err := ioutil.ReadFile(path.Join(inputDirectory, "container-job.yml"))
	if err != nil {
		t.Fatalf("error reading test file")
	}
	
	// Test: Skip container jobs when skipContainerJobs = true
	got, gotUpdated, err := AddAction(string(input), "step-security/harden-runner@v2", false, false, true)
	if err != nil {
		t.Errorf("AddAction() with skipContainerJobs=true error = %v", err)
	}
	if gotUpdated {
		t.Errorf("AddAction() with skipContainerJobs=true should not update container job")
	}
	if got != string(input) {
		t.Errorf("AddAction() with skipContainerJobs=true should not modify the yaml")
	}
}
