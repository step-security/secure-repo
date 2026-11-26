package dependabot

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	dependabotconfig "github.com/paulvollmer/dependabot-config-go"
	"gopkg.in/yaml.v3"
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
			// The existing file already has an npm entry with groups and cooldown.
			// That entry is requested again and a github-actions entry is added
			// with groups, cooldown and a value that needs quoting. The npm entry
			// is neither duplicated nor rewritten, and the whole file is still in
			// Prettier's format afterwards.
			fileName:    "prettier-append-to-groups-cooldown.yml",
			description: "existing dependabot.yml with groups and cooldown: the existing entry is kept and the new one is appended in the same format",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "npm", Directory: "/", Interval: "weekly"},
				{
					PackageEcosystem: "github-actions",
					Directory:        "/",
					Interval:         "weekly",
					CommitMessage:    &dependabotconfig.CommitMessage{Prefix: "[CI]", Include: "scope"},
					Groups: map[string]Group{
						"all-github-actions": {AppliesTo: "version-updates", Patterns: []string{"*"}},
					},
					CoolDown: &CoolDown{DefaultDays: 5, Include: []string{"*"}},
				},
			},
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
			// Whatever was already in the file is never rewritten: the result
			// starts with the input, byte for byte.
			if !strings.HasPrefix(output.FinalOutput, strings.TrimRight(string(input), "\n")) {
				t.Errorf("existing content was modified\n--- input ---\n%s\n--- got ---\n%s", input, output.FinalOutput)
			}
		})
	}
}

// doubleQuoteQuotedStrings only changes string scalars that need quoting, at
// any depth, and leaves everything else as it is. Encoding an entry marks
// such strings with no style, but a node can also arrive already single-quoted.
func TestDoubleQuoteQuotedStrings(t *testing.T) {
	doubleQuoteQuotedStrings(nil) // must not panic

	needsQuotes := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "*"}
	singleQuoted := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "actions/*", Style: yaml.SingleQuotedStyle}
	plain := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "weekly"}
	number := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "5"}
	root := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
		{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "patterns"},
			{Kind: yaml.SequenceNode, Content: []*yaml.Node{needsQuotes, singleQuoted}},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "interval"},
			plain,
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "default-days"},
			number,
		}},
	}}

	doubleQuoteQuotedStrings(root)

	if needsQuotes.Style != yaml.DoubleQuotedStyle {
		t.Errorf("%q needs quoting: style = %v, want double-quoted", needsQuotes.Value, needsQuotes.Style)
	}
	if singleQuoted.Style != yaml.DoubleQuotedStyle {
		t.Errorf("single-quoted %q: style = %v, want double-quoted", singleQuoted.Value, singleQuoted.Style)
	}
	if plain.Style != 0 {
		t.Errorf("%q needs no quoting: style = %v, want plain", plain.Value, plain.Style)
	}
	if number.Style != 0 {
		t.Errorf("non-string %q: style = %v, want unchanged", number.Value, number.Style)
	}
}
