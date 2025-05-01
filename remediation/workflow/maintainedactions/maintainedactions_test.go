package maintainedactions

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

// WriteYAML writes the given string content to a YAML file with the specified filename.
func WriteYAML(filename string, content string) error {
	// Create or truncate the file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	// Write the string content to the file
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
	}

	return nil
}

func TestReplaceActions(t *testing.T) {
	const inputDirectory = "../../../testfiles/maintainedactions/input"
	const outputDirectory = "../../../testfiles/maintainedactions/output"

	// Activate httpmock
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock GitHub API responses for getting latest releases
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/action-semantic-pull-request/releases/latest",
		httpmock.NewStringResponder(200, `{
			"tag_name": "v5.5.5",
			"name": "v5.5.5",
			"body": "Release notes",
			"created_at": "2023-01-01T00:00:00Z"
		}`))

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/skip-duplicate-actions/releases/latest",
		httpmock.NewStringResponder(200, `{
			"tag_name": "v5.3.2",
			"name": "v5.3.2",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read input file
			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading input file: %v", err)
			}

			// Call ReplaceActions
			got, updated, err := ReplaceActions(string(input))

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceActions() error = %v, wantErr %v", err, tt.wantErr)
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
				WriteYAML(tt.outputFile+"second", got)
				t.Errorf("ReplaceActions() = %v, want %v", got, string(expectedOutput))
			}
		})
	}
}
