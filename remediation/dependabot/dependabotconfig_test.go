package dependabot

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"testing"

	dependabotconfig "github.com/paulvollmer/dependabot-config-go"
)

func intPtr(i int) *int    { return &i }
func boolPtr(b bool) *bool { return &b }

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
		{
			// Additive — add github-actions with all library fields (assignees, reviewers,
			// labels, milestone, commit-message, allow, ignore, etc.) to a config that
			// already has npm. Comments, blank lines, and registries block preserved.
			inputFileName:  "additive-library-fields.yml",
			outputFileName: "additive-library-fields.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem:      "github-actions",
					Directory:             "/",
					Interval:              "weekly",
					Assignees:             []string{"ci-bot"},
					Reviewers:             []string{"platform-team"},
					Labels:                []string{"ci", "github-actions"},
					Milestone:             intPtr(2),
					OpenPullRequestsLimit: intPtr(3),
					CommitMessage: &dependabotconfig.CommitMessage{
						Prefix:  "[CI]",
						Include: "scope",
					},
					RebaseStrategy:        "auto",
					VersioningStrategy:    "increase",
					TargetBranch:          "main",
					PullRequestBranchName: &dependabotconfig.PullRequestBranchName{Separator: "/"},
					Allow:                 []dependabotconfig.Allow{{DependencyName: "actions/*"}},
					Ignore:                []dependabotconfig.Ignore{{DependencyName: "actions/checkout", Versions: []string{">= 5"}}},
				},
			},
			isChanged: true,
		},
		{
			// Additive — add npm with all extended fields (registries, exclude-paths,
			// vendor, insecure-external-code-execution, multi-ecosystem-group,
			// enable-beta-ecosystems, cooldown) to a config that already has pip.
			// Comments, blank lines, and registries block preserved.
			inputFileName:  "additive-extended-fields.yml",
			outputFileName: "additive-extended-fields.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem:              "npm",
					Directory:                     "/frontend",
					Interval:                      "daily",
					Registries:                    []string{"github-npm"},
					ExcludePaths:                  []string{"node_modules/*", ".cache/*"},
					Vendor:                        boolPtr(false),
					InsecureExternalCodeExecution: "deny",
					MultiEcosystemGroup:           "frontend-deps",
					EnableBetaEcosystems:          boolPtr(true),
					CoolDown:                      &CoolDown{DefaultDays: 5, SemverMajorDays: 14},
				},
			},
			isChanged: true,
		},
		{
			// Additive — add github-actions with directories (plural), labels, groups,
			// and open-pull-requests-limit to a monorepo config that already has npm
			// with directories. Comments, blank lines, and registries block preserved.
			inputFileName:  "additive-directories.yml",
			outputFileName: "additive-directories.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem:      "github-actions",
					Directory:             "/",
					Interval:              "weekly",
					Directories:           []string{"/", "/.github"},
					Labels:                []string{"ci"},
					OpenPullRequestsLimit: intPtr(5),
					Groups: map[string]Group{
						"actions": {Patterns: []string{"actions/*"}},
					},
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
			// Subtractive — update all library-supported fields: scalars (interval,
			// rebase-strategy, target-branch, versioning-strategy, milestone,
			// open-pull-requests-limit), string lists (assignees, reviewers, labels),
			// commit-message sub-fields, pull-request-branch-name separator,
			// schedule sub-fields (day, time, timezone), and object lists (allow, ignore).
			fileName: "subtractive-library-fields.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Interval:         "weekly",
					Day:              "friday",
					Time:             "14:00",
					Timezone:         "Europe/London",
					Allow: []dependabotconfig.Allow{
						{DependencyName: "react"},
						{DependencyName: "angular", DependencyType: "development"},
					},
					Assignees: []string{"user3", "user4", "user5"},
					CommitMessage: &dependabotconfig.CommitMessage{
						Prefix:            "chore",
						PrefixDevelopment: "build",
					},
					Ignore: []dependabotconfig.Ignore{
						{DependencyName: "jquery", Versions: []string{"3.x"}},
					},
					Labels:                []string{"deps", "automated"},
					Milestone:             intPtr(10),
					OpenPullRequestsLimit: intPtr(5),
					PullRequestBranchName: &dependabotconfig.PullRequestBranchName{Separator: "-"},
					RebaseStrategy:        "disabled",
					Reviewers:             []string{"lead-dev"},
					TargetBranch:          "main",
					VersioningStrategy:    "lockfile-only",
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — add new library-supported fields to a minimal config
			// that only has package-ecosystem, directory, and schedule.interval.
			fileName: "subtractive-add-library-fields.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem:      "npm",
					Directory:             "/",
					RebaseStrategy:        "auto",
					TargetBranch:          "develop",
					VersioningStrategy:    "increase",
					Milestone:             intPtr(3),
					OpenPullRequestsLimit: intPtr(7),
					Assignees:             []string{"dev1", "dev2"},
					Reviewers:             []string{"lead"},
					Labels:                []string{"deps"},
					CommitMessage:         &dependabotconfig.CommitMessage{Prefix: "deps"},
					PullRequestBranchName: &dependabotconfig.PullRequestBranchName{Separator: "/"},
					Allow:                 []dependabotconfig.Allow{{DependencyName: "lodash"}},
					Ignore:                []dependabotconfig.Ignore{{DependencyName: "webpack", Versions: []string{"5.x"}}},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — add schedule sub-fields (day, time, timezone) to a
			// config that only has schedule.interval.
			fileName: "subtractive-schedule-subfields.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Day:              "wednesday",
					Time:             "10:00",
					Timezone:         "Asia/Kolkata",
				},
			},
			isChanged: true,
		},
		{
			// Test updating all 6 ExtendedUpdate-only fields in-place.
			fileName: "subtractive-extended-fields.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem:              "npm",
					Directory:                     "/",
					Registries:                    []string{"npm-private", "github-registry"},
					ExcludePaths:                  []string{"dist/*", "build/*"},
					Vendor:                        boolPtr(false),
					InsecureExternalCodeExecution: "deny",
					MultiEcosystemGroup:           "updated-group",
					EnableBetaEcosystems:          boolPtr(true),
				},
			},
			isChanged: true,
		},
		{
			// Test adding all 6 ExtendedUpdate-only fields to a minimal config.
			fileName: "subtractive-add-extended-fields.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem:              "npm",
					Directory:                     "/",
					Registries:                    []string{"npm-private", "github-registry"},
					ExcludePaths:                  []string{"dist/*", "build/*"},
					Vendor:                        boolPtr(false),
					InsecureExternalCodeExecution: "deny",
					MultiEcosystemGroup:           "updated-group",
					EnableBetaEcosystems:          boolPtr(true),
				},
			},
			isChanged: true,
		},
		{
			// Realistic multi-ecosystem config: updates 3 existing ecosystems
			// (bundler, docker, github-actions) with a mix of scalar, list, block,
			// and boolean field changes, and adds a brand-new npm ecosystem.
			// Verifies comments, blank lines, and top-level registries are preserved.
			fileName: "subtractive-complex-multi-ecosystem.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "bundler",
					Directory:        "/manager",
					Interval:         "weekly",
					Day:              "monday",
					CoolDown: &CoolDown{
						DefaultDays:     3,
						SemverMajorDays: 14,
						SemverMinorDays: 5,
					},
					InsecureExternalCodeExecution: "deny",
					Labels:                        []string{"dependabot-gem-upgrade", "security"},
					OpenPullRequestsLimit:         intPtr(0),
					CommitMessage: &dependabotconfig.CommitMessage{
						Prefix:  "[DEPS] ",
						Include: "scope",
					},
					TargetBranch: "develop",
					Vendor:       boolPtr(true),
				},
				{
					PackageEcosystem:      "docker",
					Directory:             "/.github",
					Interval:              "weekly",
					Assignees:             []string{"infra-team", "devops-lead"},
					Reviewers:             []string{"platform-team"},
					OpenPullRequestsLimit: intPtr(3),
					RebaseStrategy:        "auto",
				},
				{
					PackageEcosystem:      "github-actions",
					Directory:             "/",
					OpenPullRequestsLimit: intPtr(5),
					CommitMessage: &dependabotconfig.CommitMessage{
						Prefix:  "[CI] ",
						Include: "scope",
					},
					TargetBranch: "main",
					Ignore: []dependabotconfig.Ignore{
						{DependencyName: "actions/checkout", Versions: []string{">= 5"}},
					},
				},
				{
					PackageEcosystem:      "npm",
					Directory:             "/frontend",
					Interval:              "weekly",
					Labels:                []string{"dependabot-npm-upgrade"},
					OpenPullRequestsLimit: intPtr(5),
					CommitMessage: &dependabotconfig.CommitMessage{
						Prefix:  "[DEPS] ",
						Include: "scope",
					},
					Ignore: []dependabotconfig.Ignore{
						{DependencyName: "typescript", Versions: []string{"5.x"}},
					},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — update an existing directories list (plural).
			fileName: "subtractive-directories-update.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/frontend",
					Directories:      []string{"/frontend", "/backend", "/shared"},
				},
			},
			isChanged: true,
		},
		{
			// Subtractive — add directories to a config that only has directory (singular).
			fileName: "subtractive-directories-add.yml",
			ecosystems: []Ecosystem{
				{
					PackageEcosystem: "npm",
					Directory:        "/",
					Directories:      []string{"/frontend", "/backend"},
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
