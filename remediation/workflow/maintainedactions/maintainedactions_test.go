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

	// Resolving each original action's major tag to the concrete version it points
	// at: major tag -> commit SHA -> concrete tag on that commit.
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/amannn/action-semantic-pull-request/commits/refs/tags/v5",
		httpmock.NewStringResponder(200, `sha-amannn-v5`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/amannn/action-semantic-pull-request/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v5.5.3","object":{"sha":"sha-amannn-v5","type":"commit"}}
		]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/fkirc/skip-duplicate-actions/commits/refs/tags/v5",
		httpmock.NewStringResponder(200, `sha-fkirc-v5`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/fkirc/skip-duplicate-actions/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v5.3.1","object":{"sha":"sha-fkirc-v5","type":"commit"}}
		]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/chetan/git-restore-mtime-action/commits/refs/tags/v1",
		httpmock.NewStringResponder(200, `sha-chetan-v1`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/chetan/git-restore-mtime-action/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v1.3.0","object":{"sha":"sha-chetan-v1","type":"commit"}}
		]`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/tespkg/actions-cache/commits/refs/tags/v1",
		httpmock.NewStringResponder(200, `sha-tespkg-v1`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/tespkg/actions-cache/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v1.8.0","object":{"sha":"sha-tespkg-v1","type":"commit"}}
		]`))

	// The forks' major tags resolve to the same versions, so replacing is not a
	// downgrade. (Both sides must be mocked, otherwise the downgrade check errors
	// out and silently falls back to replacing.)
	mockForkMajorTagVersion("step-security/action-semantic-pull-request", "v5", "v5.5.3")
	mockForkMajorTagVersion("step-security/skip-duplicate-actions", "v5", "v5.3.1")
	mockForkMajorTagVersion("step-security/git-restore-mtime-action", "v1", "v1.3.0")
	mockForkMajorTagVersion("step-security/actions-cache", "v1", "v1.8.0")

	tests := []struct {
		name        string
		inputFile   string
		outputFile  string
		wantUpdated bool
		wantErr     bool
	}{
		{
			name:        "one job with actions to replace",
			inputFile:   "oneJob_majorTag.yml",
			outputFile:  "oneJob_majorTag.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "no changes needed - already using maintained actions",
			inputFile:   "noChangesNeeded_majorTag.yml",
			outputFile:  "noChangesNeeded_majorTag.yml",
			wantUpdated: false,
			wantErr:     false,
		},
		{
			name:        "double job with actions to replace",
			inputFile:   "doubleJob_majorTag.yml",
			outputFile:  "doubleJob_majorTag.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "composite action with actions to replace",
			inputFile:   "compositeAction_majorTag.yml",
			outputFile:  "compositeAction_majorTag.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "no replacement when fork does not have matching major version",
			inputFile:   "noMatchingMajorVersion_majorTag.yml",
			outputFile:  "noMatchingMajorVersion_majorTag.yml",
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
			inputFile:   "oneJob_latest.yml",
			outputFile:  "oneJob_latest.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "no changes needed - already using maintained actions",
			inputFile:   "noChangesNeeded_latest.yml",
			outputFile:  "noChangesNeeded_latest.yml",
			wantUpdated: false,
			wantErr:     false,
		},
		{
			name:        "double job with latest release versions",
			inputFile:   "doubleJob_latest.yml",
			outputFile:  "doubleJob_latest.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "composite action with latest release versions",
			inputFile:   "compositeAction_latest.yml",
			outputFile:  "compositeAction_latest.yml",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "replacement happens even when major version differs (latest release used)",
			inputFile:   "noMatchingMajorVersion_latest.yml",
			outputFile:  "noMatchingMajorVersion_latest.yml",
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