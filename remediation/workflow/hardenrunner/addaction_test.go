package hardenrunner

import (
	"io/ioutil"
	"path"
	"strings"
	"testing"

	metadata "github.com/step-security/secure-repo/remediation/workflow/metadata"
	"gopkg.in/yaml.v3"
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
		{name: "anchored and aliased steps", args: args{inputYaml: "anchored-steps.yml"}, want: "anchored-steps.yml", wantErr: false, wantUpdated: true},
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
		{
			name:        "subtractive all jobs already have custom action: no changes, no commit",
			inputFile:   "customActionAllJobsPresent.yml",
			config:      HardenRunnerConfig{Config: customConfig, Subtractive: true},
			wantUpdated: false,
			outputFile:  "customActionAllJobsPresent.yml",
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
		{
			name:      "subtractive config already matches pinned action: no changes, no commit",
			inputFile: "updateConfigAlreadyMatches.yml",
			config: HardenRunnerConfig{
				Config:      "- name: Harden the runner\n  uses: step-security/harden-runner@ab7a9404c0f3da075243ca237b5fac12c98deaa5 # v2.19.3\n  with:\n    use-policy-store: true\n    api-key: ${{ secrets.STEPSECURITY_POLICY_STORE_API_KEY }}",
				Subtractive: true,
			},
			wantUpdated: false,
			outputFile:  "updateConfigAlreadyMatches.yml",
		},
		{
			name:      "subtractive takes new name and preserves pinned SHA",
			inputFile: "updateConfigNameChange.yml",
			config: HardenRunnerConfig{
				Config:      "- name: Harden the runner\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block",
				Subtractive: true,
			},
			wantUpdated: true,
			outputFile:  "updateConfigNameChange.yml",
		},
		{
			name:      "subtractive preserves pinned SHA when with: params change",
			inputFile: "updateConfigPolicyStore.yml",
			config: HardenRunnerConfig{
				Config:      "- name: Harden the runner\n  uses: step-security/harden-runner@v2\n  with:\n    use-policy-store: true\n    api-key: ${{ secrets.NEW_POLICY_STORE_KEY }}",
				Subtractive: true,
			},
			wantUpdated: true,
			outputFile:  "updateConfigPolicyStore.yml",
		},
		{
			name:      "subtractive preserves pinned SHA for step with no name key",
			inputFile: "updateConfigNoName.yml",
			config: HardenRunnerConfig{
				Config:      "- uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block",
				Subtractive: true,
			},
			wantUpdated: true,
			outputFile:  "updateConfigNoName.yml",
		},
		{
			name:      "subtractive uses config tag when action path changes",
			inputFile: "updateConfigActionPathChange.yml",
			config: HardenRunnerConfig{
				Config:      "- name: Harden the runner\n  uses: step-security/composite-runner@v2\n  with:\n    egress-policy: block",
				Subtractive: true,
			},
			wantUpdated: true,
			outputFile:  "updateConfigActionPathChange.yml",
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
		{
			name:      "subtractive with label: matching job preserves pinned SHA, non-matching job unchanged",
			inputFile: "labelSubtractiveTagPreserve.yml",
			config: HardenRunnerConfig{
				Config:           "- name: Harden the runner\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block",
				Subtractive:      true,
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "labelSubtractiveTagPreserve.yml",
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

func TestExemptRunnerLabels(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"

	tests := []struct {
		name        string
		inputFile   string
		config      HardenRunnerConfig
		wantUpdated bool
		outputFile  string
		unchanged   bool // if true, output should equal input (harden-runner not added)
	}{
		{
			name:      "exact exempt match skips harden-runner",
			inputFile: "labelScalar.yml", // runs-on: ubuntu-latest
			config:    HardenRunnerConfig{ExemptRunnerLabels: []string{"ubuntu-latest"}},
			unchanged: true,
		},
		{
			name:      "wildcard exempt match skips harden-runner",
			inputFile: "labelScalar.yml",
			config:    HardenRunnerConfig{ExemptRunnerLabels: []string{"ubuntu-*"}},
			unchanged: true,
		},
		{
			name:      "case-insensitive exempt match skips harden-runner",
			inputFile: "labelScalar.yml",
			config:    HardenRunnerConfig{ExemptRunnerLabels: []string{"UBUNTU-LATEST"}},
			unchanged: true,
		},
		{
			name:      "exempt matches a label in the runs-on array",
			inputFile: "labelArray.yml",
			config:    HardenRunnerConfig{ExemptRunnerLabels: []string{"ubuntu-latest"}},
			unchanged: true,
		},
		{
			name:        "exempt no match still adds harden-runner",
			inputFile:   "labelScalar.yml",
			config:      HardenRunnerConfig{ExemptRunnerLabels: []string{"windows-*"}},
			wantUpdated: true,
			outputFile:  "labelScalar.yml",
		},
		{
			name:      "exempt takes precedence over the runner-labels allow-list",
			inputFile: "labelScalar.yml",
			config: HardenRunnerConfig{
				SkipHardenRunner:   true,
				RunnerLabels:       []string{"ubuntu-latest"}, // allow-list would add HR
				ExemptRunnerLabels: []string{"ubuntu-*"},      // but exempt wins → skipped
			},
			unchanged: true,
		},
		{
			name:        "multi-job fixture: exempt runner skipped, non-exempt job gets harden-runner",
			inputFile:   "exemptRunnerLabels.yml",
			config:      HardenRunnerConfig{ExemptRunnerLabels: []string{"gpu-*"}},
			wantUpdated: true,
			outputFile:  "exemptRunnerLabels.yml",
		},
		{
			// Scenario 1: exempt list is present but nothing matches any job's
			// runner, so harden-runner is added to every job by default.
			name:        "exempt list present but no job matches adds harden-runner to all jobs",
			inputFile:   "exemptNoMatchMultiJob.yml",
			config:      HardenRunnerConfig{ExemptRunnerLabels: []string{"windows-*"}},
			wantUpdated: true,
			outputFile:  "exemptNoMatchMultiJob.yml",
		},
		{
			// Scenario 2: matching any single label in a runs-on list (here the
			// middle label, not the first) skips that job, while the other jobs
			// whose runners do not match still get harden-runner added.
			name:        "exempt matches one of several runs-on labels skips only that job",
			inputFile:   "exemptMatchArrayLabel.yml", // build: [self-hosted, linux, arm64]; lint/package: non-matching
			config:      HardenRunnerConfig{ExemptRunnerLabels: []string{"linux"}},
			wantUpdated: true,
			outputFile:  "exemptMatchArrayLabel.yml",
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
					t.Errorf("AddAction() expected no changes (exempt) but output differs from input\nGot:\n%s\nWant:\n%s", got, string(input))
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

	tests := []struct {
		name      string
		inputFile string
	}{
		{
			name:      "mapping style container skipped",
			inputFile: "container-job.yml",
		},
		{
			name:      "scalar style container skipped",
			inputFile: "container-job-scalar.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading input file: %v", err)
			}
			output, err := ioutil.ReadFile(path.Join(outputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading output file: %v", err)
			}

			got, gotUpdated, err := AddAction(string(input), HardenRunnerConfig{Config: defaultTestConfig}, false, false, true)
			if err != nil {
				t.Errorf("AddAction() with skipContainerJobs=true error = %v", err)
			}
			if gotUpdated {
				t.Errorf("AddAction() with skipContainerJobs=true should not update container job")
			}
			if got != string(output) {
				t.Errorf("AddAction() with skipContainerJobs=true should not modify the yaml")
			}
		})
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

func TestUpdateHardenRunnerConfigComprehensive(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"

	customConfig := "- name: Harden the runner with custom action\n  uses: acme-corp/harden-runner@v2\n  with:\n    egress-policy: block"
	hrConfig := "- name: Harden the runner\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block"

	tests := []struct {
		name        string
		inputFile   string
		config      HardenRunnerConfig
		wantUpdated bool
		outputFile  string
	}{
		{
			name:      "custom config: hr→custom uses config tag, custom→custom preserves SHA, label mismatch skipped, no-action job gets step added",
			inputFile: "customActionMultiJob.yml",
			config: HardenRunnerConfig{
				Config:           customConfig,
				Subtractive:      true,
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "customActionMultiJob.yml",
		},
		{
			name:      "hr config: build SHA preserved, test gets hr added alongside custom, label mismatch skipped, lint gets hr added",
			inputFile: "hrConfigMultiJob.yml",
			config: HardenRunnerConfig{
				Config:           hrConfig,
				Subtractive:      true,
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: true,
			outputFile:  "hrConfigMultiJob.yml",
		},
		{
			name:      "custom config: all jobs already match, no update",
			inputFile: "customActionMultiJobMatches.yml",
			config: HardenRunnerConfig{
				Config:           customConfig,
				Subtractive:      true,
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: false,
			outputFile:  "customActionMultiJobMatches.yml",
		},
		{
			name:      "hr config: all jobs already match, no update",
			inputFile: "hrConfigMultiJobMatches.yml",
			config: HardenRunnerConfig{
				Config:           hrConfig,
				Subtractive:      true,
				SkipHardenRunner: true,
				RunnerLabels:     []string{"ubuntu-latest"},
			},
			wantUpdated: false,
			outputFile:  "hrConfigMultiJobMatches.yml",
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

// Regression tests for YAML anchor/alias handling. A workflow whose jobs share
// steps via `steps: &anchor` / `steps: *anchor` used to be corrupted by the
// line-splice insert (the alias node carries no !!seq tag, so the line-based
// lookup matched a different job's steps), and the resulting parse error blanked
// the whole file in the generated PR.
func TestAddActionAnchoredAliasedSteps(t *testing.T) {
	input := `name: ci
on: push
jobs:
  build:
    runs-on: macos-latest
    steps: &build_steps
      - name: Checkout
        uses: actions/checkout@v4
      - name: Build
        run: make build
  build-reproducible:
    runs-on: macos-latest
    steps: *build_steps
  build-linux:
    runs-on: ubuntu-latest
    container: ubuntu:20.04
    steps: &container_build_steps
      - name: Checkout
        uses: actions/checkout@v4
      - name: Build
        run: make build
  build-linux-reproducible:
    runs-on: ubuntu-latest
    container: ubuntu:20.04
    steps: *container_build_steps
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test
        run: make test
`
	out, updated, err := AddAction(input, HardenRunnerConfig{Config: defaultTestConfig}, false, false, false)
	if err != nil {
		t.Fatalf("AddAction() error = %v, want nil", err)
	}
	if !updated {
		t.Fatalf("AddAction() updated = false, want true")
	}
	if strings.TrimSpace(out) == "" {
		t.Fatalf("AddAction() returned empty output")
	}

	// Output must still be valid YAML.
	workflow := metadata.Workflow{}
	if err := yaml.Unmarshal([]byte(out), &workflow); err != nil {
		t.Fatalf("AddAction() output is not valid YAML: %v", err)
	}

	// Every job must have harden-runner after alias resolution (alias jobs
	// inherit it from their anchor job).
	for jobName, job := range workflow.Jobs {
		found := false
		for _, step := range job.Steps {
			if strings.HasPrefix(step.Uses, HardenRunnerActionPath) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("job %s does not have harden-runner after AddAction()", jobName)
		}
	}

	// The alias lines themselves must be untouched (no direct insert into
	// alias jobs) and harden-runner must appear exactly once per steps block:
	// two anchors + one plain job = 3 inserts.
	if !strings.Contains(out, "steps: *build_steps") || !strings.Contains(out, "steps: *container_build_steps") {
		t.Errorf("alias steps lines were modified:\n%s", out)
	}
	if got := strings.Count(out, "uses: step-security/harden-runner"); got != 3 {
		t.Errorf("harden-runner inserted %d times, want 3:\n%s", got, out)
	}
}

// A job whose steps use flow style cannot be line-spliced safely; it must be
// skipped without corrupting the rest of the workflow.
func TestAddActionFlowStyleStepsSkipped(t *testing.T) {
	input := `name: ci
on: push
jobs:
  flow:
    runs-on: ubuntu-latest
    steps: [{name: Test, run: make test}]
  block:
    runs-on: ubuntu-latest
    steps:
      - name: Test
        run: make test
`
	out, updated, err := AddAction(input, HardenRunnerConfig{Config: defaultTestConfig}, false, false, false)
	if err != nil {
		t.Fatalf("AddAction() error = %v, want nil", err)
	}
	if !updated {
		t.Fatalf("AddAction() updated = false, want true (block job should be updated)")
	}
	workflow := metadata.Workflow{}
	if err := yaml.Unmarshal([]byte(out), &workflow); err != nil {
		t.Fatalf("AddAction() output is not valid YAML: %v", err)
	}
	if got := strings.Count(out, "uses: step-security/harden-runner"); got != 1 {
		t.Errorf("harden-runner inserted %d times, want 1 (flow job skipped):\n%s", got, out)
	}
}

// Subtractive config updates must also be anchor/alias safe: the anchor job is
// updated in place and the alias job is left untouched.
func TestUpdateHardenRunnerConfigAnchoredSteps(t *testing.T) {
	input := `name: ci
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps: &build_steps
      - name: Harden the runner (Audit all outbound calls)
        uses: step-security/harden-runner@v2
        with:
          egress-policy: audit
      - name: Checkout
        uses: actions/checkout@v4
  build-reproducible:
    runs-on: ubuntu-latest
    steps: *build_steps
`
	config := HardenRunnerConfig{
		Config:      "- name: Harden the runner\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block",
		Subtractive: true,
	}
	out, updated, err := AddAction(input, config, false, false, false)
	if err != nil {
		t.Fatalf("AddAction() error = %v, want nil", err)
	}
	if !updated {
		t.Fatalf("AddAction() updated = false, want true")
	}
	workflow := metadata.Workflow{}
	if err := yaml.Unmarshal([]byte(out), &workflow); err != nil {
		t.Fatalf("AddAction() output is not valid YAML: %v", err)
	}
	if got := strings.Count(out, "egress-policy: block"); got != 1 {
		t.Errorf("updated config appears %d times, want 1:\n%s", got, out)
	}
	if strings.Contains(out, "egress-policy: audit") {
		t.Errorf("old config still present:\n%s", out)
	}
	if !strings.Contains(out, "steps: *build_steps") {
		t.Errorf("alias steps line was modified:\n%s", out)
	}
}

// Errors must never blank the output: on any failure AddAction returns the
// input unchanged, not an empty string.
func TestAddActionInvalidYamlReturnsInput(t *testing.T) {
	input := "name: ci\njobs:\n  build:\n    steps:\n  bad_indent: [unclosed"
	out, updated, err := AddAction(input, HardenRunnerConfig{Config: defaultTestConfig}, false, false, false)
	if err == nil {
		t.Fatalf("AddAction() error = nil, want parse error")
	}
	if updated {
		t.Errorf("AddAction() updated = true, want false")
	}
	if out != input {
		t.Errorf("AddAction() on invalid yaml returned %q, want the input unchanged", out)
	}
}

// TestHardenRunnerUpdateBoundary covers the reported issue where a subtractive update
// of an already-present harden-runner step produced no-value PRs: a comment sitting
// between the harden-runner step and the next step was deleted, or a blank line was
// added where none existed. The step region must be bounded by the step's own content
// only, so trailing blank lines and comments (which belong to the next step) are always
// preserved, and re-applying an identical config is a no-op.
func TestHardenRunnerUpdateBoundary(t *testing.T) {
	const inputDirectory = "../../../testfiles/addaction/input"
	const outputDirectory = "../../../testfiles/addaction/output"

	// Configs that already match the workflow's step (used for the no-op cases).
	policyStoreCfg := "- name: Harden Runner\n  uses: step-security/harden-runner@v2\n  with:\n    use-policy-store: true\n    api-key: ${{ secrets.STEP_SECURITY_API_KEY }}"
	auditCfg := "- name: Harden Runner\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: audit"
	// Config that differs (audit -> block), so the step is genuinely updated.
	blockCfg := "- name: Harden Runner\n  uses: step-security/harden-runner@v2\n  with:\n    egress-policy: block"

	tests := []struct {
		name        string
		inputFile   string
		config      HardenRunnerConfig
		wantUpdated bool
		outputFile  string
	}{
		{
			// Identical config, trailing blank + comment before the next step:
			// must be a no-op and the comment must survive.
			name:        "identical config with trailing comment is a no-op",
			inputFile:   "hrUpdatePreserveComment.yml",
			config:      HardenRunnerConfig{Config: policyStoreCfg, Subtractive: true},
			wantUpdated: false,
			outputFile:  "hrUpdatePreserveComment.yml",
		},
		{
			// Identical config, next step immediately after (no blank line):
			// must be a no-op and must not add a stray blank line.
			name:        "identical config with no blank before next step is a no-op",
			inputFile:   "hrUpdateNoBlankBeforeNext.yml",
			config:      HardenRunnerConfig{Config: auditCfg, Subtractive: true},
			wantUpdated: false,
			outputFile:  "hrUpdateNoBlankBeforeNext.yml",
		},
		{
			// Changed config with a trailing comment: the step updates but the
			// comment and blank line that belong to the next step are preserved.
			name:        "changed config preserves the trailing comment",
			inputFile:   "hrUpdateChangedPreservesComment.yml",
			config:      HardenRunnerConfig{Config: blockCfg, Subtractive: true},
			wantUpdated: true,
			outputFile:  "hrUpdateChangedPreservesComment.yml",
		},
		{
			// A comment inside the step's own body is replaced along with the step
			// (it is re-rendered from config), while the trailing comment survives.
			name:        "comment inside the step body does not truncate the step",
			inputFile:   "hrUpdateCommentInsideStep.yml",
			config:      HardenRunnerConfig{Config: blockCfg, Subtractive: true},
			wantUpdated: true,
			outputFile:  "hrUpdateCommentInsideStep.yml",
		},
		{
			// Multi-job: the job that already has harden-runner (matching config) is
			// left untouched with its trailing comment preserved, while the other job
			// has no harden-runner and must get it added as its first step.
			name:        "multi-job: existing job unchanged with comment, other job gets harden-runner added",
			inputFile:   "hrUpdateMultiJobMixed.yml",
			config:      HardenRunnerConfig{Config: auditCfg, Subtractive: true},
			wantUpdated: true,
			outputFile:  "hrUpdateMultiJobMixed.yml",
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
				t.Fatalf("AddAction() error = %v", err)
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
