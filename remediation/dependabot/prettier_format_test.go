package dependabot

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	dependabotconfig "github.com/paulvollmer/dependabot-config-go"
)

// Entries StepSecurity writes into dependabot.yml must already be in the format
// Prettier produces with its default options: 2-space indentation at every
// level, including inside groups and cooldown, and double-quoted strings such
// as "*". Repositories that run Prettier in CI otherwise fail on the generated
// file.
//
// Each case is one fixture name: testfiles/dependabotfiles/input/<name> is the
// existing dependabot.yml (empty when the repository has none) and
// testfiles/dependabotfiles/output/<name> is the expected result, a file that
// `prettier --check` accepts.
func TestNewDependabotEntriesMatchPrettierFormat(t *testing.T) {
	const inputDirectory = "../../testfiles/dependabotfiles/input"
	const outputDirectory = "../../testfiles/dependabotfiles/output"

	githubActionsWithGroupAndCooldown := Ecosystem{
		PackageEcosystem: "github-actions",
		Directory:        "/",
		Interval:         "weekly",
		Groups: map[string]Group{
			"all-github-actions": {AppliesTo: "version-updates", Patterns: []string{"*"}},
		},
		CoolDown: &CoolDown{DefaultDays: 5, Include: []string{"*"}},
	}

	tests := []struct {
		fileName    string
		ecosystems  []Ecosystem
		description string
	}{
		{
			fileName:    "prettier-new-file-groups-cooldown.yml",
			description: "no dependabot.yml yet: the file is built from scratch with groups and cooldown",
			ecosystems: []Ecosystem{
				githubActionsWithGroupAndCooldown,
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "daily",
					Groups: map[string]Group{
						"all-npm": {AppliesTo: "version-updates", Patterns: []string{"*"}},
					},
					CoolDown: &CoolDown{DefaultDays: 3, Include: []string{"*"}},
				},
			},
		},
		{
			fileName:    "prettier-append-groups-cooldown.yml",
			description: "existing dependabot.yml: a new ecosystem with groups and cooldown is appended",
			ecosystems:  []Ecosystem{githubActionsWithGroupAndCooldown},
		},
		{
			fileName:    "prettier-new-file-quoted-values.yml",
			description: "values that need quoting are double-quoted",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "github-actions",
					Directory:        "/",
					Interval:         "weekly",
					CommitMessage:    &dependabotconfig.CommitMessage{Prefix: "[CI]", Include: "scope"},
					Ignore:           []dependabotconfig.Ignore{{DependencyName: "actions/checkout", Versions: []string{">= 5"}}},
					Groups: map[string]Group{
						"actions": {Patterns: []string{"actions/*"}},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.fileName, func(t *testing.T) {
			input, err := os.ReadFile(path.Join(inputDirectory, test.fileName))
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			body, err := json.Marshal(UpdateDependabotConfigRequest{Content: string(input), Ecosystems: test.ecosystems})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}

			output, err := UpdateDependabotConfig(string(body))
			if err != nil {
				t.Fatalf("UpdateDependabotConfig: %v", err)
			}
			if !output.IsChanged {
				t.Fatalf("%s: expected a change", test.description)
			}

			expected, err := os.ReadFile(path.Join(outputDirectory, test.fileName))
			if err != nil {
				t.Fatalf("read expected output: %v", err)
			}
			if output.FinalOutput != string(expected) {
				t.Errorf("%s\n--- got ---\n%s\n--- want ---\n%s", test.description, output.FinalOutput, expected)
			}
		})
	}
}
