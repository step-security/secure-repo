package hardenrunner

import (
	"io/ioutil"
	"path"
	"testing"
)

const defaultTestConfig = "- name: Harden the runner (Audit all outbound calls)\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: audit"

func TestAddAction(t *testing.T) {
	type args struct {
		inputYaml string
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
		{name: "one job", args: args{inputYaml: "action-issues.yml"}, want: "action-issues.yml", wantErr: false, wantUpdated: true},
		{name: "two jobs", args: args{inputYaml: "2jobs.yml"}, want: "2jobs.yml", wantErr: false, wantUpdated: true},
		{name: "already present", args: args{inputYaml: "alreadypresent.yml"}, want: "alreadypresent.yml", wantErr: false, wantUpdated: true},
		{name: "already present 2", args: args{inputYaml: "alreadypresent_2.yml"}, want: "alreadypresent_2.yml", wantErr: false, wantUpdated: false},
		{name: "reusable job", args: args{inputYaml: "reusablejob.yml"}, want: "reusablejob.yml", wantErr: false, wantUpdated: false},
		{name: "job name in input", args: args{inputYaml: "jobNameInInput.yml"}, want: "jobNameInInput.yml", wantErr: false, wantUpdated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.args.inputYaml))
			if err != nil {
				t.Fatalf("error reading test file")
			}
			got, gotUpdated, err := AddAction(string(input), HardenRunnerConfig{Config: defaultTestConfig}, false, false, false)

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

func TestUpdateHardenRunnerConfig(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"

	blockConfig := "- name: Harden the runner (Audit all outbound calls)\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block\n    allowed-endpoints: >\n      github.com:443\n      api.github.com:443"

	tests := []struct {
		name        string
		config      HardenRunnerConfig
		wantUpdated bool
		outputFile  string
	}{
		{
			name:        "subtractive true replaces existing config",
			config:      HardenRunnerConfig{Config: blockConfig, Subtractive: true},
			wantUpdated: true,
			outputFile:  "updateConfig.yml",
		},
		{
			name:        "subtractive false does not change existing config",
			config:      HardenRunnerConfig{Config: blockConfig, Subtractive: false},
			wantUpdated: false,
			outputFile:  "updateConfigNotSubtractive.yml",
		},
	}

	input, err := ioutil.ReadFile(path.Join(inputDirectory, "updateConfig.yml"))
	if err != nil {
		t.Fatalf("error reading input file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotUpdated, err := AddAction(string(input), tt.config, false, false, false)
			if err != nil {
				t.Errorf("AddAction() error = %v", err)
			}
			if gotUpdated != tt.wantUpdated {
				t.Errorf("AddAction() updated = %v, wantUpdated %v", gotUpdated, tt.wantUpdated)
			}
			expected, err := ioutil.ReadFile(path.Join(outputDirectory, tt.outputFile))
			if err != nil {
				t.Fatalf("error reading output file: %v", err)
			}
			if got != string(expected) {
				t.Errorf("AddAction() = %v, want %v", got, string(expected))
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
	got, gotUpdated, err := AddAction(string(input), HardenRunnerConfig{Config: defaultTestConfig}, false, false, true)
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
