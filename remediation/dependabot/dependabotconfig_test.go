package dependabot

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"testing"
)

func TestConfigDependabotFile(t *testing.T) {

	const inputDirectory = "../../testfiles/dependabotfiles/input"
	const outputDirectory = "../../testfiles/dependabotfiles/output"

	tests := []struct {
		fileName   string
		Ecosystems []Ecosystem
		isChanged  bool
	}{
		{
			fileName:   "Without-github-action.yml",
			Ecosystems: []Ecosystem{{"github-actions", "/", "daily", nil, nil}, {"npm", "/app", "daily", nil, nil}},
			isChanged:  true,
		},
		{
			fileName:   "With-github-action.yml",
			Ecosystems: []Ecosystem{{"github-actions", "/", "daily", nil, nil}},
			isChanged:  false,
		},
		{
			fileName:   "File-not-exit.yml",
			Ecosystems: []Ecosystem{{"github-actions", "/", "daily", nil, nil}},
			isChanged:  true,
		},
		{
			fileName:   "Same-ecosystem-different-directory.yml",
			Ecosystems: []Ecosystem{{"github-actions", "/", "daily", nil, nil}, {"npm", "/sample", "daily", nil, nil}},
			isChanged:  true,
		},
		{
			fileName:   "No-Indentation.yml",
			Ecosystems: []Ecosystem{{"npm", "/sample", "daily", nil, nil}},
			isChanged:  true,
		},
		{
			fileName:   "High-Indentation.yml",
			Ecosystems: []Ecosystem{{"npm", "/sample", "daily", nil, nil}},
			isChanged:  true,
		},
		{
			fileName:   "extra-slash.yml",
			Ecosystems: []Ecosystem{{"npm", "/sample", "daily", nil, nil}},
			isChanged:  false,
		},
	}

	for _, test := range tests {
		var updateDependabotConfigRequest UpdateDependabotConfigRequest
		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}
		updateDependabotConfigRequest.Content = string(input)
		updateDependabotConfigRequest.Ecosystems = test.Ecosystems
		inputRequest, err := json.Marshal(updateDependabotConfigRequest)
		if err != nil {
			log.Fatal(err)
		}

		output, err := UpdateDependabotConfig(string(inputRequest))
		if err != nil {
			t.Fatalf("Error not expected: %s", err)
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}

		if string(expectedOutput) != output.FinalOutput {
			t.Errorf("test failed %s did not match expected output\n%s", test.fileName, output.FinalOutput)
		}

		if output.IsChanged != test.isChanged {
			t.Errorf("test failed %s did not match IsChanged, Expected: %v Got: %v", test.fileName, test.isChanged, output.IsChanged)

		}

	}

}

func TestGroups(t *testing.T) {
	const inputDirectory = "../../testfiles/dependabotfiles/input"
	const outputDirectory = "../../testfiles/dependabotfiles/output"

	tests := []struct {
		inputFileName  string
		outputFileName string
		ecosystems     []Ecosystem
		subtractive    bool
		isChanged      bool
	}{
		{
			// Subtractive — group exists with multiple fields, only patterns updated, other fields unchanged.
			inputFileName:  "group-prs-modify-patterns-only.yml",
			outputFileName: "group-prs-modify-patterns-only.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Groups: map[string]Group{
						"all": {Patterns: []string{"*"}},
					},
				},
			},
			subtractive: true,
			isChanged:   true,
		},
		{
			// Subtractive — entry has one group, new group with multiple attributes added alongside.
			inputFileName:  "group-prs-add-new-group.yml",
			outputFileName: "group-prs-add-new-group.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Groups: map[string]Group{
						"dev-deps": {
							DependencyType: "development",
							UpdateTypes:    []string{"minor", "patch"},
							Patterns:       []string{"eslint*", "jest*"},
						},
					},
				},
			},
			subtractive: true,
			isChanged:   true,
		},
		{
			// Subtractive — all slice fields (patterns, exclude-patterns, update-types) updated;
			// dependency-type string left unchanged since it is not specified.
			inputFileName:  "group-prs-modify-slices.yml",
			outputFileName: "group-prs-modify-slices.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Groups: map[string]Group{
						"all": {
							Patterns:        []string{"angular*", "react*"},
							ExcludePatterns: []string{"lodash", "axios"},
							UpdateTypes:     []string{"minor", "patch"},
						},
					},
				},
			},
			subtractive: true,
			isChanged:   true,
		},
		{
			// Additive (non-subtractive) — complex real-world file with registries, comments, labels, etc.;
			// adding a new npm ecosystem preserves original content exactly and appends the new entry.
			inputFileName:  "complex-multi-ecosystem.yml",
			outputFileName: "complex-multi-ecosystem-additive.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/frontend",
					Interval:         "daily",
					CoolDown:         &CoolDown{DefaultDays: 7, SemverMajorDays: 30},
				},
			},
			subtractive: false,
			isChanged:   true,
		},
		{
			// Additive (non-subtractive) — ecosystem already exists, groups not applied, output unchanged.
			inputFileName:  "group-prs-modify-patterns-only.yml",
			outputFileName: "group-prs-modify-patterns-only-no-change.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Groups: map[string]Group{
						"all": {Patterns: []string{"*"}},
					},
				},
			},
			subtractive: false,
			isChanged:   false,
		},
	}

	for _, test := range tests {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.inputFileName))
		if err != nil {
			log.Fatal(err)
		}
		req := UpdateDependabotConfigRequest{
			Content:     string(input),
			Ecosystems:  test.ecosystems,
			Subtractive: test.subtractive,
		}
		inputJSON, err := json.Marshal(req)
		if err != nil {
			log.Fatal(err)
		}

		output, err := UpdateDependabotConfig(string(inputJSON))
		if err != nil {
			t.Fatalf("Error not expected: %s", err)
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, test.outputFileName))
		if err != nil {
			log.Fatal(err)
		}

		if string(expectedOutput) != output.FinalOutput {
			t.Errorf("test failed %s did not match expected output\n%s", test.outputFileName, output.FinalOutput)
		}
		if output.IsChanged != test.isChanged {
			t.Errorf("test failed %s did not match IsChanged, Expected: %v Got: %v", test.outputFileName, test.isChanged, output.IsChanged)
		}
	}
}

func TestAdditiveCoolDown(t *testing.T) {
	const inputDirectory = "../../testfiles/dependabotfiles/input"
	const outputDirectory = "../../testfiles/dependabotfiles/output"

	tests := []struct {
		inputFileName  string
		outputFileName string
		ecosystems     []Ecosystem
		isChanged      bool
	}{
		{
			// Additive — new ecosystem added with CoolDown; CoolDown must appear in output.
			inputFileName:  "additive-new-with-cooldown.yml",
			outputFileName: "additive-new-with-cooldown.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown:         &CoolDown{DefaultDays: 5},
				},
			},
			isChanged: true,
		},
	}

	for _, test := range tests {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.inputFileName))
		if err != nil {
			log.Fatal(err)
		}
		req := UpdateDependabotConfigRequest{
			Content:     string(input),
			Ecosystems:  test.ecosystems,
			Subtractive: false,
		}
		inputJSON, err := json.Marshal(req)
		if err != nil {
			log.Fatal(err)
		}

		output, err := UpdateDependabotConfig(string(inputJSON))
		if err != nil {
			t.Fatalf("Error not expected: %s", err)
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, test.outputFileName))
		if err != nil {
			log.Fatal(err)
		}

		if string(expectedOutput) != output.FinalOutput {
			t.Errorf("test failed %s did not match expected output\n%s", test.outputFileName, output.FinalOutput)
		}
		if output.IsChanged != test.isChanged {
			t.Errorf("test failed %s did not match IsChanged, Expected: %v Got: %v", test.outputFileName, test.isChanged, output.IsChanged)
		}
	}
}

func TestUpdateSubtractiveFields(t *testing.T) {
	const inputDirectory = "../../testfiles/dependabotfiles/input"
	const outputDirectory = "../../testfiles/dependabotfiles/output"

	tests := []struct {
		fileName   string
		ecosystems []Ecosystem
		isChanged  bool
	}{
		{
			fileName: "subtractive-add-cooldown.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					CoolDown:         &CoolDown{DefaultDays: 5, SemverPatchDays: 2},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — input uses flow sequence syntax patterns: ["*"];
			// verifies that flow style is preserved when patterns are updated and
			// cooldown is added.
			fileName: "flow-sequence-syntax.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown:         &CoolDown{DefaultDays: 5},
					Groups: map[string]Group{
						"all": {Patterns: []string{"lodash", "axios"}},
					},
				},
				{
					PackageEcosystem: "pip",
					Directory:        "/backend",
					Groups: map[string]Group{
						"all": {Patterns: []string{"requests", "flask"}},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — rich file with registries, comments, cooldown (semver + include/exclude),
			// and groups with multiple slice fields. Updates npm cooldown semver days, replaces
			// include/exclude lists, updates group patterns/exclude-patterns/update-types,
			// and adds a new group. Verifies comments, registries, labels are all preserved.
			fileName: "subtractive-rich-update.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown: &CoolDown{
						SemverMajorDays: 30,
						SemverMinorDays: 14,
						SemverPatchDays: 7,
						Include:         []string{"lodash", "axios", "react"},
						Exclude:         []string{"express", "webpack"},
					},
					Groups: map[string]Group{
						"production": {
							Patterns:        []string{"react", "react-dom", "redux"},
							ExcludePatterns: []string{"lodash", "axios"},
							UpdateTypes:     []string{"minor", "patch"},
						},
						"dev-tools": {
							Patterns:    []string{"jest", "eslint", "prettier"},
							UpdateTypes: []string{"minor", "patch"},
						},
						"new-group": {
							AppliesTo:      "version-updates",
							DependencyType: "production",
							Patterns:       []string{"typescript", "ts-node"},
						},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — complex file with bundler, docker, github-actions;
			// update bundler cooldown and interval, and github-actions interval + add cooldown + group.
			// Docker entry is untouched.
			fileName: "complex-multi-ecosystem.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "bundler",
					Directory:        "/manager",
					Interval:         "weekly",
					CoolDown: &CoolDown{
						DefaultDays:     3,
						SemverMajorDays: 14,
						SemverPatchDays: 2,
					},
					Groups: map[string]Group{
						"rubocop": {Patterns: []string{"rubocop", "rubocop-rspec", "rubocop-rails", "rubocop-performance", "rubocop-minitest"}},
					},
				},
				{
					PackageEcosystem: "github-actions",
					Directory:        "/",
					Interval:         "monthly",
					CoolDown: &CoolDown{
						DefaultDays:     14,
						SemverMajorDays: 60,
					},
					Groups: map[string]Group{
						"actions": {Patterns: []string{"*"}},
					},
				},
			},
			isChanged: true,
		},
		{
			fileName: "subtractive-modify-interval-and-major.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown:         &CoolDown{SemverMajorDays: 20},
				},
			},
			isChanged: true,
		},
		{
			fileName: "subtractive-modify-include-exclude.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					CoolDown: &CoolDown{
						Include: []string{"lodash", "axios"},
						Exclude: []string{"express", "react"},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — request has two ecosystems; github-actions exists and gets updated,
			// npm does not exist in the config and is silently skipped.
			fileName: "subtractive-multi-skip-missing.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "github-actions",
					Directory:        "/",
					Interval:         "monthly",
					CoolDown: &CoolDown{
						DefaultDays:     14,
						SemverMajorDays: 60,
					},
					Groups: map[string]Group{
						"actions": {Patterns: []string{"*"}},
					},
				},
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown: &CoolDown{
						DefaultDays:     7,
						SemverMajorDays: 30,
						SemverMinorDays: 14,
						SemverPatchDays: 5,
					},
					Groups: map[string]Group{
						"production-dependencies": {
							AppliesTo:      "version-updates",
							Patterns:       []string{"*"},
							DependencyType: "production",
						},
						"dev-dependencies": {
							AppliesTo:      "version-updates",
							Patterns:       []string{"*"},
							DependencyType: "development",
						},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — cooldown fields appear in non-standard order (jumbled);
			// verifies that values are updated at the correct lines regardless of field order.
			fileName: "subtractive-jumbled-cooldown.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown: &CoolDown{
						SemverMajorDays: 30,
						SemverMinorDays: 14,
						SemverPatchDays: 7,
						Include:         []string{"lodash", "axios", "react"},
						Exclude:         []string{"express", "webpack"},
					},
				},
			},
			isChanged: true,
		},
	}

	for _, test := range tests {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}
		req := UpdateDependabotConfigRequest{
			Content:     string(input),
			Ecosystems:  test.ecosystems,
			Subtractive: true,
		}
		inputJSON, err := json.Marshal(req)
		if err != nil {
			log.Fatal(err)
		}

		output, err := UpdateDependabotConfig(string(inputJSON))
		if err != nil {
			t.Fatalf("Error not expected: %s", err)
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}

		if string(expectedOutput) != output.FinalOutput {
			t.Errorf("test failed %s did not match expected output\n%s", test.fileName, output.FinalOutput)
		}
		if output.IsChanged != test.isChanged {
			t.Errorf("test failed %s did not match IsChanged, Expected: %v Got: %v", test.fileName, test.isChanged, output.IsChanged)
		}
	}
}
