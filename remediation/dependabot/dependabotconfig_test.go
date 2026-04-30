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
			Ecosystems: []Ecosystem{{PackageEcosystem: "github-actions", Directory: "/", Interval: "daily"}, {PackageEcosystem: "npm", Directory: "/app", Interval: "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "With-github-action.yml",
			Ecosystems: []Ecosystem{{PackageEcosystem: "github-actions", Directory: "/", Interval: "daily"}},
			isChanged:  false,
		},
		{
			fileName:   "File-not-exit.yml",
			Ecosystems: []Ecosystem{{PackageEcosystem: "github-actions", Directory: "/", Interval: "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "Same-ecosystem-different-directory.yml",
			Ecosystems: []Ecosystem{{PackageEcosystem: "github-actions", Directory: "/", Interval: "daily"}, {PackageEcosystem: "npm", Directory: "/sample", Interval: "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "No-Indentation.yml",
			Ecosystems: []Ecosystem{{PackageEcosystem: "npm", Directory: "/sample", Interval: "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "High-Indentation.yml",
			Ecosystems: []Ecosystem{{PackageEcosystem: "npm", Directory: "/sample", Interval: "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "extra-slash.yml",
			Ecosystems: []Ecosystem{{PackageEcosystem: "npm", Directory: "/sample", Interval: "daily"}},
			isChanged:  false,
		},
		{
			fileName:   "npm-with-registries-and-groups.yml",
			Ecosystems: []Ecosystem{{PackageEcosystem: "github-actions", Directory: "/", Interval: "daily"}},
			isChanged:  true,
		},
		{
			fileName:   "rich-attributes-additive.yml",
			Ecosystems: []Ecosystem{{PackageEcosystem: "github-actions", Directory: "/", Interval: "daily"}},
			isChanged:  true,
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
		{
			// Additive — existing entry uses directories (plural); the requested directory is
			// already in the list → no duplicate entry created, output unchanged.
			inputFileName:  "directories-no-duplicate.yml",
			outputFileName: "directories-no-duplicate.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "github-actions", Directory: "/", Interval: "daily"},
			},
			subtractive: false,
			isChanged:   false,
		},
		{
			// Additive — existing docker entry uses directories (plural); new directory
			// not in the list → appended to existing directories instead of new entry.
			inputFileName:  "directories-append.yml",
			outputFileName: "directories-append.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "docker", Directory: "/services", Interval: "daily"},
			},
			subtractive: false,
			isChanged:   true,
		},
		{
			// Additive — two ecosystems each using directories (plural):
			//   docker: already has cooldown in config; request sends different cooldown + same interval
			//     → only /services appended, existing cooldown (default-days:5, semver-major-days:30) untouched.
			//   npm: no cooldown in config; request sends cooldown + different interval (daily vs weekly)
			//     → only /backend appended, no cooldown added, interval stays weekly.
			// Verifies that in additive mode only directories are appended; cooldown and interval
			// updates on existing entries belong to subtractive mode. Also checks that registries,
			// comments, and labels are fully preserved.
			inputFileName:  "directories-append-with-cooldown.yml",
			outputFileName: "directories-append-with-cooldown.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "docker", Directory: "/services", Interval: "daily", CoolDown: &CoolDown{DefaultDays: 10}},
				{PackageEcosystem: "npm", Directory: "/backend", Interval: "daily", CoolDown: &CoolDown{DefaultDays: 3}},
			},
			subtractive: false,
			isChanged:   true,
		},
		{
			// Additive — three docker directories appended to the same docker entry,
			// three npm directories appended to the same npm entry, plus a new pip
			// ecosystem that doesn't exist in the config (gets added as new entry).
			// Verifies collect-then-apply dedup, sorted index processing across
			// multiple entries, and new entry insertion with registries, comments,
			// and labels preserved.
			inputFileName:  "directories-append-multiple.yml",
			outputFileName: "directories-append-multiple.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "docker", Directory: "/api", Interval: "daily"},
				{PackageEcosystem: "docker", Directory: "/shared", Interval: "daily"},
				{PackageEcosystem: "docker", Directory: "/workers", Interval: "daily"},
				{PackageEcosystem: "npm", Directory: "/backend", Interval: "weekly"},
				{PackageEcosystem: "npm", Directory: "/api-client", Interval: "weekly"},
				{PackageEcosystem: "npm", Directory: "/common", Interval: "weekly"},
				{PackageEcosystem: "pip", Directory: "/backend", Interval: "weekly"},
			},
			subtractive: false,
			isChanged:   true,
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
			// Subtractive — npm entry with registries and groups at top-level after updates;
			// updates npm in-place and adds github-actions (not in config) within the updates
			// section. Verifies the registries block below updates is preserved untouched.
			fileName: "npm-with-registries-and-groups-subtractive.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown:         &CoolDown{DefaultDays: 5},
				},
				{
					PackageEcosystem: "github-actions",
					Directory:        "/",
					Interval:         "daily",
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — pip ecosystem interval change from daily to weekly;
			// cooldown with semver fields left untouched.
			fileName: "subtractive-pip-interval-change.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "pip",
					Directory:        "/",
					Interval:         "weekly",
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
		{
			// Subtractive — add include and exclude lists to an existing cooldown
			// that only has int fields; exercises buildBlockSeq.
			fileName: "subtractive-add-include-exclude.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					CoolDown: &CoolDown{
						Include: []string{"lodash", "axios"},
						Exclude: []string{"express"},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — modify applies-to, dependency-type, and group-by
			// scalar fields on an existing group.
			fileName: "group-modify-scalars.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Groups: map[string]Group{
						"production": {
							AppliesTo:      "security-updates",
							DependencyType: "development",
							GroupBy:        "dependency-name",
						},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — add missing fields (applies-to, exclude-patterns,
			// dependency-type, update-types, group-by) to an existing group that
			// only has patterns; exercises addScalar, addSeq, and buildBlockSeq
			// within applyGroupUpdates.
			fileName: "group-add-missing-fields.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Groups: map[string]Group{
						"production": {
							AppliesTo:       "version-updates",
							DependencyType:  "production",
							ExcludePatterns: []string{"lodash", "axios"},
							UpdateTypes:     []string{"minor", "patch"},
							GroupBy:         "semver-level",
						},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — input uses single-quoted YAML values; verifies that
			// replaceScalarOnLine preserves single-quote style.
			fileName: "single-quoted-values.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown:         &CoolDown{DefaultDays: 10},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — existing entry uses directories (plural); interval update
			// must match via directories and update in place, not create a duplicate entry.
			fileName: "subtractive-directories-interval.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "github-actions", Directory: "/", Interval: "weekly"},
			},
			isChanged: true,
		},
		{
			// Subtractive — existing entry uses directories (plural) but requested directory
			// is not in the list; directory must be appended and interval updated in place,
			// not create a separate new entry.
			fileName: "subtractive-directories-append.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "github-actions", Directory: "/", Interval: "weekly"},
			},
			isChanged: true,
		},
		{
			// Subtractive — existing entry uses directories (plural); requested directory
			// is already in the list; cooldown block must be added to the matched entry.
			fileName: "subtractive-directories-add-cooldown.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "github-actions", Directory: "/", CoolDown: &CoolDown{DefaultDays: 5}},
			},
			isChanged: true,
		},
		{
			// Subtractive — existing entry uses directories (plural); requested directory
			// is not in the list; directory must be appended and cooldown block added,
			// all on the existing entry.
			fileName: "subtractive-directories-append-with-cooldown.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "github-actions", Directory: "/", CoolDown: &CoolDown{DefaultDays: 5}},
			},
			isChanged: true,
		},
		{
			// Subtractive — realistic config with registries, comments, and labels; two ecosystems
			// each using directories (plural) with directory not in list:
			//   docker: already has cooldown; request sends updated cooldown + same interval
			//     → /services appended, cooldown updated (default-days:5→10, semver-major-days:30→60).
			//   npm: no cooldown; request sends cooldown + different interval (weekly→daily)
			//     → /backend appended, cooldown block added, interval changed to daily.
			// github-actions (singular directory) is untouched. Registries, comments, and labels preserved.
			fileName: "subtractive-directories-append-with-cooldown-rich.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "docker", Directory: "/services", Interval: "daily", CoolDown: &CoolDown{DefaultDays: 10, SemverMajorDays: 60}},
				{PackageEcosystem: "npm", Directory: "/backend", Interval: "daily", CoolDown: &CoolDown{DefaultDays: 3}},
			},
			isChanged: true,
		},
		{
			// Subtractive — real-world shaped config: github-actions + nuget + docker (singular
			// directories) alongside npm which uses directories (plural) with no existing cooldown.
			// github-actions uses directories (plural), / already in list → cooldown added in place.
			// npm uses directories (plural), /temp not in list → /temp appended, cooldown + groups added.
			// nuget and docker (singular directory) are left untouched.
			fileName: "subtractive-directories-append-existing-cooldown.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/temp",
					CoolDown:         &CoolDown{DefaultDays: 7, SemverMajorDays: 30, SemverMinorDays: 14, SemverPatchDays: 5},
					Interval:         "weekly",
					Groups: map[string]Group{
						"dev-dependencies":        {AppliesTo: "version-updates", Patterns: []string{"*"}, DependencyType: "development"},
						"production-dependencies": {AppliesTo: "version-updates", Patterns: []string{"*"}, DependencyType: "production"},
					},
				},
				{
					PackageEcosystem: "github-actions",
					Directory:        "/",
					Interval:         "monthly",
					CoolDown:         &CoolDown{DefaultDays: 14},
					Groups: map[string]Group{"actions": {
						Patterns: []string{"*"},
					}},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — rich config with schedule.day/time/timezone, assignees,
			// reviewers, labels, milestone, open-pull-requests-limit, target-branch,
			// commit-message, rebase-strategy, versioning-strategy; verifies all
			// attributes are preserved when updating interval, cooldown, and groups.
			fileName: "rich-attributes-subtractive.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					CoolDown:         &CoolDown{DefaultDays: 10},
					Groups: map[string]Group{
						"all": {Patterns: []string{"react", "angular"}},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — three docker directories appended to docker entry,
			// three npm directories appended to npm entry, plus a new pip ecosystem
			// (gets added). Docker interval changed from daily to weekly.
			// Verifies merged replaceSequence across multiple entries, new entry
			// insertion, and preservation of registries, comments, and labels.
			fileName: "subtractive-directories-append-multiple.yml",
			ecosystems: []Ecosystem{
				{PackageEcosystem: "docker", Directory: "/api", Interval: "weekly"},
				{PackageEcosystem: "docker", Directory: "/shared", Interval: "weekly"},
				{PackageEcosystem: "docker", Directory: "/workers", Interval: "weekly"},
				{PackageEcosystem: "npm", Directory: "/backend", Interval: "daily"},
				{PackageEcosystem: "npm", Directory: "/api-client", Interval: "daily"},
				{PackageEcosystem: "npm", Directory: "/common", Interval: "daily"},
				{PackageEcosystem: "pip", Directory: "/backend", Interval: "weekly"},
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

func TestErrorHandling(t *testing.T) {
	t.Run("invalid JSON input", func(t *testing.T) {
		_, err := UpdateDependabotConfig("not valid json")
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("invalid YAML content", func(t *testing.T) {
		req := UpdateDependabotConfigRequest{
			Content:    ":\n  :\n    - [invalid",
			Ecosystems: []Ecosystem{{PackageEcosystem: "npm", Directory: "/", Interval: "daily"}},
		}
		inputJSON, _ := json.Marshal(req)
		_, err := UpdateDependabotConfig(string(inputJSON))
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})

	t.Run("empty content with no ecosystems", func(t *testing.T) {
		req := UpdateDependabotConfigRequest{
			Content:    "",
			Ecosystems: []Ecosystem{},
		}
		inputJSON, _ := json.Marshal(req)
		output, err := UpdateDependabotConfig(string(inputJSON))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if output.IsChanged {
			t.Error("expected IsChanged to be false for empty content with no ecosystems")
		}
	})

	t.Run("subtractive with empty content", func(t *testing.T) {
		req := UpdateDependabotConfigRequest{
			Content:     "",
			Ecosystems:  []Ecosystem{{PackageEcosystem: "npm", Directory: "/", Interval: "daily"}},
			Subtractive: true,
		}
		inputJSON, _ := json.Marshal(req)
		output, err := UpdateDependabotConfig(string(inputJSON))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if output.IsChanged {
			t.Error("expected IsChanged to be false for subtractive with empty content")
		}
	})

	t.Run("additive with content missing updates key", func(t *testing.T) {
		req := UpdateDependabotConfigRequest{
			Content:    "version: 2\n",
			Ecosystems: []Ecosystem{{PackageEcosystem: "npm", Directory: "/", Interval: "daily"}},
		}
		inputJSON, _ := json.Marshal(req)
		_, err := UpdateDependabotConfig(string(inputJSON))
		if err == nil {
			t.Fatal("expected error for YAML content with no updates list")
		}
	})

	t.Run("subtractive with content missing updates key", func(t *testing.T) {
		req := UpdateDependabotConfigRequest{
			Content:     "version: 2\n",
			Ecosystems:  []Ecosystem{{PackageEcosystem: "npm", Directory: "/", Interval: "daily"}},
			Subtractive: true,
		}
		inputJSON, _ := json.Marshal(req)
		_, err := UpdateDependabotConfig(string(inputJSON))
		if err == nil {
			t.Fatal("expected error for subtractive with no updates list")
		}
	})

	t.Run("subtractive with empty ecosystems", func(t *testing.T) {
		content := "version: 2\nupdates:\n  - package-ecosystem: npm\n    directory: /\n    schedule:\n      interval: daily\n"
		req := UpdateDependabotConfigRequest{
			Content:     content,
			Ecosystems:  []Ecosystem{},
			Subtractive: true,
		}
		inputJSON, _ := json.Marshal(req)
		output, err := UpdateDependabotConfig(string(inputJSON))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if output.IsChanged {
			t.Error("expected IsChanged to be false for subtractive with empty ecosystems")
		}
		if output.FinalOutput != content {
			t.Error("expected output to match original content")
		}
	})
}
