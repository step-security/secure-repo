package maintainedactions

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestReplaceActions(t *testing.T) {
	const inputDirectory = "../../../testfiles/maintainedActions/input"
	const outputDirectory = "../../../testfiles/maintainedActions/output"

	// Activate httpmock
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock GitHub API responses for checking major version tags on forks
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/action-semantic-pull-request/git/ref/tags/v5",
		httpmock.NewStringResponder(200, `{"ref":"refs/tags/v5","object":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","type":"commit"}}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/action-semantic-pull-request/git/ref/tags/v3",
		httpmock.NewStringResponder(404, `{"message":"Not Found"}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/skip-duplicate-actions/git/ref/tags/v5",
		httpmock.NewStringResponder(200, `{"ref":"refs/tags/v5","object":{"sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","type":"commit"}}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/git-restore-mtime-action/git/ref/tags/v1",
		httpmock.NewStringResponder(200, `{"ref":"refs/tags/v1","object":{"sha":"cccccccccccccccccccccccccccccccccccccccc","type":"commit"}}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/actions-cache/git/ref/tags/v1",
		httpmock.NewStringResponder(200, `{"ref":"refs/tags/v1","object":{"sha":"dddddddddddddddddddddddddddddddddddddddd","type":"commit"}}`))

	tests := []struct {
		name        string
		inputFile   string
		outputFile  string
		wantUpdated bool
		wantErr     bool
	}{
		{
			name:        "one job with actions to replace",
			inputFile:   "oneJob.yml",
			outputFile:  "oneJob.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "no changes needed - already using maintained actions",
			inputFile:   "noChangesNeeded.yml",
			outputFile:  "noChangesNeeded.yml",
			wantUpdated: false,
			wantErr:     false,
		},
		{
			name:        "double job with actions to replace",
			inputFile:   "doubleJob.yml",
			outputFile:  "doubleJob.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "composite action with actions to replace",
			inputFile:   "compositeAction.yml",
			outputFile:  "compositeAction.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "no replacement when fork does not have matching major version",
			inputFile:   "noMatchingMajorVersion.yml",
			outputFile:  "noMatchingMajorVersion.yml",
			wantUpdated: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read input file
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading input file: %v", err)
			}
			actionMap, err := LoadMaintainedActions("maintainedActions.json")
			if err != nil {
				t.Errorf("ReplaceActions() unable to json file %v", err)
				return
			}
			got, updated, replaceErr := ReplaceActions(string(input), actionMap, true)

			// Check error
			if (replaceErr != nil) != tt.wantErr {
				t.Errorf("ReplaceActions() error = %v, wantErr %v", replaceErr, tt.wantErr)
				return
			}

			// Check if updated flag matches
			if updated != tt.wantUpdated {
				t.Errorf("ReplaceActions() updated = %v, wantUpdated %v", updated, tt.wantUpdated)
			}

			// Read expected output file
			expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, tt.outputFile))
			if err != nil {
				t.Fatalf("error reading expected output file: %v", err)
			}

			// Compare output with expected
			if got != string(expectedOutput) {
				// WriteYAML(tt.outputFile+"second", got)
				t.Errorf("ReplaceActions() = %v, want %v", got, string(expectedOutput))
			}
		})
	}
}

func TestReplaceActionsLatestRelease(t *testing.T) {
	const inputDirectory = "../../../testfiles/maintainedActions/input"
	const outputDirectory = "../../../testfiles/maintainedActions/output"

	// Activate httpmock
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock GitHub API responses for GetLatestRelease (GET /repos/{owner}/{repo}/releases/latest)
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/action-semantic-pull-request/releases/latest",
		httpmock.NewStringResponder(200, `{"id":1,"tag_name":"v6.1.0","name":"v6.1.0"}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/skip-duplicate-actions/releases/latest",
		httpmock.NewStringResponder(200, `{"id":2,"tag_name":"v5.3.1","name":"v5.3.1"}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/git-restore-mtime-action/releases/latest",
		httpmock.NewStringResponder(200, `{"id":3,"tag_name":"v2.0.0","name":"v2.0.0"}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/actions-cache/releases/latest",
		httpmock.NewStringResponder(200, `{"id":4,"tag_name":"v4.0.0","name":"v4.0.0"}`))

	tests := []struct {
		name        string
		inputFile   string
		outputFile  string
		wantUpdated bool
		wantErr     bool
	}{
		{
			name:        "one job with latest release versions",
			inputFile:   "oneJob.yml",
			outputFile:  "oneJobLatest.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "no changes needed - already using maintained actions",
			inputFile:   "noChangesNeeded.yml",
			outputFile:  "noChangesNeeded.yml",
			wantUpdated: false,
			wantErr:     false,
		},
		{
			name:        "double job with latest release versions",
			inputFile:   "doubleJob.yml",
			outputFile:  "doubleJobLatest.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "composite action with latest release versions",
			inputFile:   "compositeAction.yml",
			outputFile:  "compositeActionLatest.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "replacement happens even when major version differs (latest release used)",
			inputFile:   "noMatchingMajorVersion.yml",
			outputFile:  "noMatchingMajorVersionLatest.yml",
			wantUpdated: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read input file
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading input file: %v", err)
			}
			actionMap, err := LoadMaintainedActions("maintainedActions.json")
			if err != nil {
				t.Errorf("ReplaceActions() unable to json file %v", err)
				return
			}
			got, updated, replaceErr := ReplaceActions(string(input), actionMap, false)

			// Check error
			if (replaceErr != nil) != tt.wantErr {
				t.Errorf("ReplaceActions() error = %v, wantErr %v", replaceErr, tt.wantErr)
				return
			}

			// Check if updated flag matches
			if updated != tt.wantUpdated {
				t.Errorf("ReplaceActions() updated = %v, wantUpdated %v", updated, tt.wantUpdated)
			}

			// Read expected output file
			expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, tt.outputFile))
			if err != nil {
				t.Fatalf("error reading expected output file: %v", err)
			}

			// Compare output with expected
			if got != string(expectedOutput) {
				t.Errorf("ReplaceActions() = %v, want %v", got, string(expectedOutput))
			}
		})
	}
}

func TestSome(t *testing.T) {

	version, err := GetMajorTagFromSHA("tj-actions/changed-files", "00f80efd45353091691a96565de08f4f50c685f8")
	if err != nil {
		t.Errorf("GetMajorTagFromSHA() error = %v", err)
	}
	t.Logf("GetMajorTagFromSHA() version = %v", version)
}
