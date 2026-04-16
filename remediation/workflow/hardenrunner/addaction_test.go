package hardenrunner

import (
	"io/ioutil"
	"path"
	"testing"
)

const defaultTestConfig = DefaultHardenRunnerConfig

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

func TestCustomActionConfig(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"

	customConfig := "- name: Security Scanner\n  uses: org/security-scanner@v3\n  with:\n    mode: strict\n    scan-level: deep"

	customConfigWithEndpoints := "- name: Harden the runner\n  uses: acme-corp/harden-runner@v2\n  with:\n    egress-policy: block\n    allowed-endpoints: >\n      github.com:443\n      registry.npmjs.org:443"

	tests := []struct {
		name        string
		inputFile   string
		config      HardenRunnerConfig
		wantUpdated bool
		outputFile  string
	}{
		{
			name:        "add custom action to single job",
			inputFile:   "customAction.yml",
			config:      HardenRunnerConfig{Config: customConfig},
			wantUpdated: true,
			outputFile:  "customAction.yml",
		},
		{
			name:        "add custom action with endpoints to two jobs",
			inputFile:   "customActionTwoJobs.yml",
			config:      HardenRunnerConfig{Config: customConfigWithEndpoints},
			wantUpdated: true,
			outputFile:  "customActionTwoJobs.yml",
		},
		{
			name:        "subtractive replaces harden-runner with custom action",
			inputFile:   "customActionSubtractive.yml",
			config:      HardenRunnerConfig{Config: customConfigWithEndpoints, Subtractive: true},
			wantUpdated: true,
			outputFile:  "customActionSubtractive.yml",
		},
		{
			name:        "three jobs: custom present, harden-runner present, empty gets action",
			inputFile:   "customActionAlreadyPresent.yml",
			config:      HardenRunnerConfig{Config: customConfig},
			wantUpdated: true,
			outputFile:  "customActionAlreadyPresent.yml",
		},
		{
			name:        "subtractive three jobs: custom unchanged, harden-runner replaced, empty gets action",
			inputFile:   "customActionAlreadyPresentSubtractive.yml",
			config:      HardenRunnerConfig{Config: customConfig, Subtractive: true},
			wantUpdated: true,
			outputFile:  "customActionAlreadyPresentSubtractive.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading input file: %v", err)
			}
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
				t.Errorf("AddAction() output mismatch\nGot:\n%s\nWant:\n%s", got, string(expected))
			}
		})
	}
}

func TestUpdateHardenRunnerConfig(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"

	blockConfig := "- name: Harden the runner (Audit all outbound calls)\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block\n    allowed-endpoints: >\n      github.com:443\n      api.github.com:443"

	blockConfigWithComments := "# Harden Runner step added by StepSecurity\n- name: Harden the runner (Audit all outbound calls)\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block\n    # Approved endpoints for CI\n    allowed-endpoints: >\n      github.com:443\n      api.github.com:443\n      # npm registry\n      registry.npmjs.org:443"

	tests := []struct {
		name        string
		inputFile   string
		config      HardenRunnerConfig
		wantUpdated bool
		outputFile  string
	}{
		{
			name:        "subtractive true replaces existing config",
			inputFile:   "updateConfig.yml",
			config:      HardenRunnerConfig{Config: blockConfig, Subtractive: true},
			wantUpdated: true,
			outputFile:  "updateConfig.yml",
		},
		{
			name:        "subtractive false does not change existing config",
			inputFile:   "updateConfig.yml",
			config:      HardenRunnerConfig{Config: blockConfig, Subtractive: false},
			wantUpdated: false,
			outputFile:  "updateConfigNotSubtractive.yml",
		},
		{
			name:        "subtractive replaces existing allowed-endpoints",
			inputFile:   "updateConfigReplaceEndpoints.yml",
			config:      HardenRunnerConfig{Config: blockConfig, Subtractive: true},
			wantUpdated: true,
			outputFile:  "updateConfigReplaceEndpoints.yml",
		},
		{
			name:        "subtractive replaces config with comments",
			inputFile:   "updateConfigWithComments.yml",
			config:      HardenRunnerConfig{Config: blockConfig, Subtractive: true},
			wantUpdated: true,
			outputFile:  "updateConfigWithComments.yml",
		},
		{
			name:        "subtractive replaces single-line allowed-endpoints",
			inputFile:   "updateConfigSingleLine.yml",
			config:      HardenRunnerConfig{Config: blockConfig, Subtractive: true},
			wantUpdated: true,
			outputFile:  "updateConfigSingleLine.yml",
		},
		{
			name:        "subtractive with comments in config",
			inputFile:   "updateConfigWithConfigComments.yml",
			config:      HardenRunnerConfig{Config: blockConfigWithComments, Subtractive: true},
			wantUpdated: true,
			outputFile:  "updateConfigWithConfigComments.yml",
		},
		{
			name:        "subtractive replaces harden-runner as last step",
			inputFile:   "updateConfigLastStep.yml",
			config:      HardenRunnerConfig{Config: blockConfig, Subtractive: true},
			wantUpdated: true,
			outputFile:  "updateConfigLastStep.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading input file: %v", err)
			}
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

func TestRunnerLabelFiltering(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"

	tests := []struct {
		name        string
		inputFile   string
		config      HardenRunnerConfig
		wantUpdated bool
		outputFile  string
		unchanged   bool // if true, output should equal input
	}{
		{
			name:      "label matches scalar",
			inputFile: "labelScalar.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "labelScalar.yml",
		},
		{
			name:      "label does not match",
			inputFile: "labelNoMatch.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: false,
			unchanged:   true,
		},
		{
			name:      "label matches in array",
			inputFile: "labelArray.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "labelArray.yml",
		},
		{
			name:      "label no match in array",
			inputFile: "labelArrayNoMatch.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: false,
			unchanged:   true,
		},
		{
			name:      "skip disabled ignores labels",
			inputFile: "labelNoMatch.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: false,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "labelNoMatch-skipDisabled.yml",
		},
		{
			name:      "empty labels list does not filter",
			inputFile: "labelScalar.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{},
			},
			wantUpdated: true,
			outputFile:  "labelScalar.yml",
		},
		{
			name:      "both slices overlap",
			inputFile: "labelArray.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"windows-latest", "ubuntu-latest", "macos-latest"},
			},
			wantUpdated: true,
			outputFile:  "labelArray.yml",
		},
		{
			name:      "both slices no overlap",
			inputFile: "labelArray.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"windows-latest", "macos-latest"},
			},
			wantUpdated: false,
			unchanged:   true,
		},
		{
			name:      "multi-job mixed labels",
			inputFile: "labelMultiJob.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "labelMultiJob.yml",
		},
		{
			name:      "mapping with labels array",
			inputFile: "labelMapping.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "labelMapping.yml",
		},
		{
			name:      "mapping with labels scalar",
			inputFile: "labelMappingScalar.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "labelMappingScalar.yml",
		},
		{
			name:      "mapping with labels no match",
			inputFile: "labelMapping.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"windows-latest"},
			},
			wantUpdated: false,
			unchanged:   true,
		},
		{
			name:      "mapping with group only no labels key",
			inputFile: "labelMappingNoLabels.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: false,
			unchanged:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading input file: %v", err)
			}
			got, gotUpdated, err := AddAction(string(input), tt.config, false, false, false)
			if err != nil {
				t.Errorf("AddAction() error = %v", err)
			}
			if gotUpdated != tt.wantUpdated {
				t.Errorf("AddAction() updated = %v, wantUpdated %v", gotUpdated, tt.wantUpdated)
			}
			if tt.unchanged {
				if got != string(input) {
					t.Errorf("AddAction() expected no changes but output differs from input\nGot:\n%s\nWant:\n%s", got, string(input))
				}
			} else if tt.outputFile != "" {
				expected, err := ioutil.ReadFile(path.Join(outputDirectory, tt.outputFile))
				if err != nil {
					t.Fatalf("error reading output file: %v", err)
				}
				if got != string(expected) {
					t.Errorf("AddAction() output mismatch\nGot:\n%s\nWant:\n%s", got, string(expected))
				}
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

func TestGetActionFromConfig(t *testing.T) {
	tests := []struct {
		name   string
		config HardenRunnerConfig
		want   string
	}{
		{
			name:   "extracts uses from config",
			config: HardenRunnerConfig{Config: "- name: Harden Runner\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: audit"},
			want:   "step-security/harden-runner@v2",
		},
		{
			name:   "extracts custom action path",
			config: HardenRunnerConfig{Config: "- name: Custom Runner\n  uses: my-org/custom-runner@v1\n  with:\n    mode: strict"},
			want:   "my-org/custom-runner@v1",
		},
		{
			name:   "falls back when no uses line",
			config: HardenRunnerConfig{Config: "- name: Harden Runner\n  run: echo hello"},
			want:   HardenRunnerActionPath,
		},
		{
			name:   "falls back on empty config",
			config: HardenRunnerConfig{Config: ""},
			want:   HardenRunnerActionPath,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getActionFromConfig(tt.config)
			if got != tt.want {
				t.Errorf("getActionFromConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddActionWithEmptyConfig(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"

	input, err := ioutil.ReadFile(path.Join(inputDirectory, "labelScalar.yml"))
	if err != nil {
		t.Fatalf("error reading test file: %v", err)
	}
	// Empty Config should use DefaultHardenRunnerConfig
	got, gotUpdated, err := AddAction(string(input), HardenRunnerConfig{}, false, false, false)
	if err != nil {
		t.Fatalf("AddAction() error = %v", err)
	}
	if !gotUpdated {
		t.Error("AddAction() expected updated = true")
	}
	expected, err := ioutil.ReadFile(path.Join(outputDirectory, "labelScalar.yml"))
	if err != nil {
		t.Fatalf("error reading output file: %v", err)
	}
	if got != string(expected) {
		t.Errorf("AddAction() with empty config mismatch\nGot:\n%s\nWant:\n%s", got, string(expected))
	}
}
