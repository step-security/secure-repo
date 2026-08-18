package maintainedactions

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

// TestReplaceActions_DowngradeGuard covers the rule that a replacement must never
// move a workflow to an older version of the action. Each case mocks the version
// the original is pinned to and the version the fork's major tag resolves to, and
// asserts the workflow is only rewritten when the fork is not behind.
func TestReplaceActions_DowngradeGuard(t *testing.T) {
	const inputDirectory = "../../../testfiles/maintainedActions/input"
	const outputDirectory = "../../../testfiles/maintainedActions/output"

	// The tag listings below carry a handful of releases on different commits, as
	// a real repository would, so the lookups have to pick out the tag that is
	// actually on the commit in question rather than the only one on offer.

	// mockUpstreamTagForSHA: the SHA the workflow pins resolves to this version.
	mockUpstreamTagForSHA := func(repo, sha, version string) {
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/"+repo+"/git/matching-refs/tags/v",
			httpmock.NewStringResponder(200, `[
				{"ref":"refs/tags/v0.4.0","object":{"sha":"upstreamold1","type":"commit"}},
				{"ref":"refs/tags/v0.5.7","object":{"sha":"upstreamold2","type":"commit"}},
				{"ref":"refs/tags/`+version+`","object":{"sha":"`+sha+`","type":"commit"}}
			]`))
	}

	// mockUpstreamMajorTagVersion: the major tag the workflow pins currently
	// points at this version (major tag -> commit -> concrete tag).
	mockUpstreamMajorTagVersion := func(repo, majorTag, version string) {
		base := "https://api.github.com/repos/" + repo
		httpmock.RegisterResponder("GET", base+"/commits/refs/tags/"+majorTag,
			httpmock.NewStringResponder(200, `upstreamsha`))
		httpmock.RegisterResponder("GET", base+"/git/matching-refs/tags/v",
			httpmock.NewStringResponder(200, `[
				{"ref":"refs/tags/v1.0.0","object":{"sha":"upstreamold1","type":"commit"}},
				{"ref":"refs/tags/v2.0.0","object":{"sha":"upstreamold2","type":"commit"}},
				{"ref":"refs/tags/`+majorTag+`","object":{"sha":"upstreamsha","type":"commit"}},
				{"ref":"refs/tags/`+version+`","object":{"sha":"upstreamsha","type":"commit"}}
			]`))
	}

	// mockForkMajorTag: the fork has the major tag (the gate) and it resolves to
	// forkVersion. An empty forkVersion makes that resolution fail.
	mockForkMajorTag := func(forkRepo, majorTag, forkVersion string) {
		base := "https://api.github.com/repos/" + forkRepo
		httpmock.RegisterResponder("GET", base+"/git/ref/tags/"+majorTag,
			httpmock.NewStringResponder(200,
				`{"ref":"refs/tags/`+majorTag+`","object":{"sha":"forksha","type":"commit"}}`))
		if forkVersion == "" {
			httpmock.RegisterResponder("GET", base+"/commits/refs/tags/"+majorTag,
				httpmock.NewStringResponder(500, `{"message":"boom"}`))
			return
		}
		httpmock.RegisterResponder("GET", base+"/commits/refs/tags/"+majorTag,
			httpmock.NewStringResponder(200, `forksha`))
		httpmock.RegisterResponder("GET", base+"/git/matching-refs/tags/v",
			httpmock.NewStringResponder(200, `[
				{"ref":"refs/tags/v0.1.0","object":{"sha":"forkold1","type":"commit"}},
				{"ref":"refs/tags/v0.2.0","object":{"sha":"forkold2","type":"commit"}},
				{"ref":"refs/tags/`+majorTag+`","object":{"sha":"forksha","type":"commit"}},
				{"ref":"refs/tags/`+forkVersion+`","object":{"sha":"forksha","type":"commit"}}
			]`))
	}

	const zizmorSHA = "3dc1ecc9bcb9e94e9b2c709687979e1298497054"

	const semanticPR = "amannn/action-semantic-pull-request"
	const semanticPRFork = "step-security/action-semantic-pull-request"

	// A case with no outputFile expects the workflow to come back untouched.
	tests := []struct {
		name       string
		inputFile  string
		outputFile string
		actionMap  map[string]string
		setupMocks func()
	}{
		{
			// The reported zizmor case: the workflow is on v0.6.2 via a SHA pin
			// while the fork's v0 is still on v0.5.7. Replacing would downgrade it.
			name:      "sha pinned newer than fork is not replaced",
			inputFile: "shaPinned_majorTag.yml",
			actionMap: map[string]string{"zizmorcore/zizmor-action": "step-security/zizmor-action"},
			setupMocks: func() {
				mockUpstreamTagForSHA("zizmorcore/zizmor-action", "3dc1ecc9bcb9e94e9b2c709687979e1298497054", "v0.6.2")
				mockForkMajorTag("step-security/zizmor-action", "v0", "v0.5.7")
			},
		},
		{
			// "@v3" currently points at v3.0.2, but the fork's v3 is on v3.0.1.
			// A major-only comparison would have missed this.
			name:      "floating major pointing at newer version is not replaced",
			inputFile: "floatingMajorPin_majorTag.yml",
			actionMap: map[string]string{"dorny/paths-filter": "step-security/paths-filter"},
			setupMocks: func() {
				mockUpstreamMajorTagVersion("dorny/paths-filter", "v3", "v3.0.2")
				mockForkMajorTag("step-security/paths-filter", "v3", "v3.0.1")
			},
		},
		{
			name:      "concrete version newer than fork is not replaced",
			inputFile: "concreteVersionPin_majorTag.yml",
			actionMap: map[string]string{semanticPR: semanticPRFork},
			setupMocks: func() {
				mockForkMajorTag(semanticPRFork, "v5", "v5.1.0")
			},
		},
		{
			// The fork's version cannot be determined, so the replacement is
			// skipped rather than risking a downgrade.
			name:      "fork version that cannot be determined is not replaced",
			inputFile: "concreteVersionPin_majorTag.yml",
			actionMap: map[string]string{semanticPR: semanticPRFork},
			setupMocks: func() {
				mockForkMajorTag(semanticPRFork, "v5", "")
			},
		},
		{
			// The fork never tagged v5.5.3 but its v5 is on the newer v5.6.0, so
			// this must still be replaced — an exact-version match would not.
			name:       "fork on a newer version is replaced",
			inputFile:  "concreteVersionPin_majorTag.yml",
			outputFile: "concreteVersionPin_majorTag.yml",
			actionMap:  map[string]string{semanticPR: semanticPRFork},
			setupMocks: func() {
				mockForkMajorTag(semanticPRFork, "v5", "v5.6.0")
			},
		},
		{
			name:       "fork on the same version is replaced",
			inputFile:  "concreteVersionPin_majorTag.yml",
			outputFile: "concreteVersionPin_majorTag.yml",
			actionMap:  map[string]string{semanticPR: semanticPRFork},
			setupMocks: func() {
				mockForkMajorTag(semanticPRFork, "v5", "v5.5.3")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			tt.setupMocks()

			input, err := ioutil.ReadFile(path.Join(inputDirectory, tt.inputFile))
			if err != nil {
				t.Fatalf("error reading input file: %v", err)
			}

			// No output file means the workflow must be left exactly as it was.
			want := string(input)
			if tt.outputFile != "" {
				expected, err := ioutil.ReadFile(path.Join(outputDirectory, tt.outputFile))
				if err != nil {
					t.Fatalf("error reading expected output file: %v", err)
				}
				want = string(expected)
			}

			got, updated, err := ReplaceActions(string(input), tt.actionMap, true)
			if err != nil {
				t.Fatalf("ReplaceActions() unexpected error: %v", err)
			}
			if wantUpdated := tt.outputFile != ""; updated != wantUpdated {
				t.Errorf("ReplaceActions() updated = %v, want %v", updated, wantUpdated)
			}
			if got != want {
				t.Errorf("ReplaceActions() output mismatch\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

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
	// at: major tag -> commit SHA -> concrete tag on that commit. Each listing also
	// carries older releases on other commits, as a real repository would.
	mockUpstreamMajorTag := func(repo, majorTag, sha, version string) {
		base := "https://api.github.com/repos/" + repo
		httpmock.RegisterResponder("GET", base+"/commits/refs/tags/"+majorTag,
			httpmock.NewStringResponder(200, sha))
		httpmock.RegisterResponder("GET", base+"/git/matching-refs/tags/v",
			httpmock.NewStringResponder(200, `[
				{"ref":"refs/tags/v0.9.0","object":{"sha":"`+sha+`-old1","type":"commit"}},
				{"ref":"refs/tags/v1.0.0","object":{"sha":"`+sha+`-old2","type":"commit"}},
				{"ref":"refs/tags/`+majorTag+`","object":{"sha":"`+sha+`","type":"commit"}},
				{"ref":"refs/tags/`+version+`","object":{"sha":"`+sha+`","type":"commit"}}
			]`))
	}

	mockUpstreamMajorTag("amannn/action-semantic-pull-request", "v5", "sha-amannn-v5", "v5.5.3")
	mockUpstreamMajorTag("fkirc/skip-duplicate-actions", "v5", "sha-fkirc-v5", "v5.3.1")
	mockUpstreamMajorTag("chetan/git-restore-mtime-action", "v1", "sha-chetan-v1", "v1.3.0")
	mockUpstreamMajorTag("tespkg/actions-cache", "v1", "sha-tespkg-v1", "v1.8.0")

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