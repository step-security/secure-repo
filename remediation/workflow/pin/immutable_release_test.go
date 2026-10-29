package pin

import (
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestIsImmutableReleaseAction(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/locked/releases/tags/v1.2.3",
		httpmock.NewStringResponder(200, `{"tag_name":"v1.2.3","immutable":true}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/mutable/releases/tags/v1.2.3",
		httpmock.NewStringResponder(200, `{"tag_name":"v1.2.3","immutable":false}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/norelease/releases/tags/v1",
		httpmock.NewStringResponder(http.StatusNotFound, `{"message":"Not Found"}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/broken/releases/tags/v1.0.0",
		httpmock.NewStringResponder(http.StatusInternalServerError, `{"message":"boom"}`))

	tests := []struct {
		action  string
		want    bool
		wantErr bool
	}{
		{action: "test-org/locked@v1.2.3", want: true},
		{action: "test-org/locked/sub/path@v1.2.3", want: true},
		{action: "test-org/mutable@v1.2.3", want: false},
		{action: "test-org/norelease@v1", want: false},
		{action: "test-org/broken@v1.0.0", wantErr: true},
		{action: "test-org/locked@11bd71901bbe5b1630ceea73d27597364c9af683", want: false},
		{action: "./local-action", want: false},
		{action: "docker://alpine:3.20", want: false},
		{action: "test-org/locked", want: false},
	}
	for _, tt := range tests {
		got, err := IsImmutableReleaseAction(tt.action)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("IsImmutableReleaseAction(%q) = %t, %v, want %t, error %t", tt.action, got, err, tt.want, tt.wantErr)
		}
	}

	// Each lookup calls the API: nothing is cached. SHA, local and docker
	// references never call it.
	if n := httpmock.GetCallCountInfo()["GET https://api.github.com/repos/test-org/locked/releases/tags/v1.2.3"]; n != 2 {
		t.Errorf("test-org/locked@v1.2.3 looked up %d times, want 2 (once per reference)", n)
	}
	if n := httpmock.GetTotalCallCount(); n != 5 {
		t.Errorf("total API calls = %d, want 5", n)
	}
}

// TestPinActions_SkipImmutableReleaseTags pins each input file in
// testfiles/pinactions/input and compares it with the output file of the same
// name. With the option on (skipimmutablereleasetags.yml), only tags that
// belong to an immutable GitHub release stay as written; a mutable release, a
// tag with no release, and a failed release lookup are all pinned. With it off
// (donotskipimmutablereleasetags.yml), everything is pinned and no release is
// looked up.
func TestPinActions_SkipImmutableReleaseTags(t *testing.T) {
	const inputDirectory = "../../../testfiles/pinactions/input"
	const outputDirectory = "../../../testfiles/pinactions/output"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/locked/releases/tags/v1.2.3",
		httpmock.NewStringResponder(200, `{"tag_name":"v1.2.3","immutable":true}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/mutable/releases/tags/v2.0.0",
		httpmock.NewStringResponder(200, `{"tag_name":"v2.0.0","immutable":false}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/norelease/releases/tags/v3.1.0",
		httpmock.NewStringResponder(http.StatusNotFound, `{"message":"Not Found"}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/broken/releases/tags/v1.0.0",
		httpmock.NewStringResponder(http.StatusInternalServerError, `{"message":"boom"}`))

	// Commits come from the map, so the release lookups are the only API calls.
	actionCommitMap := map[string]string{
		"test-org/locked@v1.2.3":          "4444444444444444444444444444444444444444",
		"test-org/locked/sub/path@v1.2.3": "4444444444444444444444444444444444444444",
		"test-org/mutable@v2.0.0":         "1111111111111111111111111111111111111111",
		"test-org/norelease@v3.1.0":       "2222222222222222222222222222222222222222",
		"test-org/broken@v1.0.0":          "3333333333333333333333333333333333333333",
	}

	tests := []struct {
		fileName        string
		skip            bool
		wantLookupCalls int
	}{
		// locked (twice: top level and sub-path), mutable, norelease, broken.
		{fileName: "skipimmutablereleasetags.yml", skip: true, wantLookupCalls: 5},
		{fileName: "donotskipimmutablereleasetags.yml", skip: false, wantLookupCalls: 0},
	}
	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			httpmock.ZeroCallCounters()

			input, err := os.ReadFile(path.Join(inputDirectory, tt.fileName))
			if err != nil {
				t.Fatal(err)
			}

			output, updated, err := PinActions(string(input), []string{"test-org/exempted"}, false, actionCommitMap, WithSkipImmutableReleaseTags(tt.skip))
			if err != nil {
				t.Fatalf("PinActions() error = %v", err)
			}
			if !updated {
				t.Errorf("PinActions() updated = false, want true")
			}

			want, err := os.ReadFile(path.Join(outputDirectory, tt.fileName))
			if err != nil {
				t.Fatal(err)
			}
			if output != string(want) {
				t.Errorf("output did not match %s\n%s", tt.fileName, output)
			}

			if n := httpmock.GetTotalCallCount(); n != tt.wantLookupCalls {
				t.Errorf("release lookups = %d, want %d", n, tt.wantLookupCalls)
			}
		})
	}
}
