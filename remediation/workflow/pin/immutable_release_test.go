package pin

import (
	"net/http"
	"os"
	"path"
	"strings"
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
// looked up. A reference in actionCommitMap is pinned to its mapped SHA even
// when its tag is an immutable release, with no release lookup
// (skipimmutablereleasetagswithmap.yml).
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

	// References not in actionCommitMap resolve their commit from GitHub. No tag
	// list is returned, so the version comment is the tag as written.
	for ref, sha := range map[string]string{
		"locked/commits/v1.2.3":    "4444444444444444444444444444444444444444",
		"mutable/commits/v2.0.0":   "1111111111111111111111111111111111111111",
		"norelease/commits/v3.1.0": "2222222222222222222222222222222222222222",
		"broken/commits/v1.0.0":    "3333333333333333333333333333333333333333",
	} {
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/"+ref, httpmock.NewStringResponder(200, sha))
	}
	httpmock.RegisterResponder("GET", `=~^https://api\.github\.com/repos/test-org/[^/]+/git/matching-refs/`,
		httpmock.NewStringResponder(200, `[]`))

	// releaseLookups counts only the release lookups, not the commit lookups.
	releaseLookups := func() int {
		n := 0
		for call, count := range httpmock.GetCallCountInfo() {
			if strings.Contains(call, "/releases/tags/") {
				n += count
			}
		}
		return n
	}

	tests := []struct {
		name            string
		fileName        string
		skip            bool
		cache           bool
		actionCommitMap map[string]string
		wantLookupCalls int
	}{
		// locked (twice: top level and sub-path), mutable, norelease, broken.
		{name: "on", fileName: "skipimmutablereleasetags.yml", skip: true, wantLookupCalls: 5},
		// The sub-path action shares test-org/locked@v1.2.3 with the top-level one.
		{name: "on with a cache", fileName: "skipimmutablereleasetags.yml", skip: true, cache: true, wantLookupCalls: 4},
		{name: "off", fileName: "donotskipimmutablereleasetags.yml", skip: false, wantLookupCalls: 0},
		// test-org/locked@v1.2.3 and test-org/mutable@v2.0.0 are mapped, so they
		// are pinned to their mapped SHAs without a release lookup. The unmapped
		// sub-path reference to the same immutable tag is the only one looked up.
		{name: "on, mapped references always pinned", fileName: "skipimmutablereleasetagswithmap.yml", skip: true, actionCommitMap: map[string]string{
			"test-org/locked@v1.2.3":  "5555555555555555555555555555555555555555",
			"test-org/mutable@v2.0.0": "6666666666666666666666666666666666666666",
		}, wantLookupCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.ZeroCallCounters()

			input, err := os.ReadFile(path.Join(inputDirectory, tt.fileName))
			if err != nil {
				t.Fatal(err)
			}

			opts := []Option{WithSkipImmutableReleaseTags(tt.skip)}
			if tt.cache {
				opts = append(opts, WithImmutableReleaseCache(NewImmutableReleaseCache()))
			}
			output, updated, err := PinActions(string(input), []string{"test-org/exempted"}, false, tt.actionCommitMap, opts...)
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

			if n := releaseLookups(); n != tt.wantLookupCalls {
				t.Errorf("release lookups = %d, want %d", n, tt.wantLookupCalls)
			}
		})
	}
}

// TestImmutableReleaseCache checks that a cache shared across lookups calls the
// API once per owner/repo@tag, that sub-path actions and differently cased owners
// share an entry, that a tag with no release is remembered too, and that a failed
// lookup is not remembered.
func TestImmutableReleaseCache(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/locked/releases/tags/v1.2.3",
		httpmock.NewStringResponder(200, `{"tag_name":"v1.2.3","immutable":true}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/Test-Org/Locked/releases/tags/v1.2.3",
		httpmock.NewStringResponder(200, `{"tag_name":"v1.2.3","immutable":true}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/norelease/releases/tags/v1",
		httpmock.NewStringResponder(http.StatusNotFound, `{"message":"Not Found"}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-org/broken/releases/tags/v1.0.0",
		httpmock.NewStringResponder(http.StatusInternalServerError, `{"message":"boom"}`))

	cache := NewImmutableReleaseCache()
	for _, tt := range []struct {
		action  string
		want    bool
		wantErr bool
	}{
		{action: "test-org/locked@v1.2.3", want: true},
		{action: "test-org/locked@v1.2.3", want: true},
		{action: "test-org/locked/sub/path@v1.2.3", want: true},
		{action: "Test-Org/Locked@v1.2.3", want: true},
		{action: "test-org/norelease@v1", want: false},
		{action: "test-org/norelease@v1", want: false},
		{action: "test-org/broken@v1.0.0", wantErr: true},
		{action: "test-org/broken@v1.0.0", wantErr: true},
	} {
		got, err := IsImmutableReleaseAction(tt.action, WithImmutableReleaseCache(cache))
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("IsImmutableReleaseAction(%q) = %t, %v, want %t, error %t", tt.action, got, err, tt.want, tt.wantErr)
		}
	}

	calls := httpmock.GetCallCountInfo()
	if n := calls["GET https://api.github.com/repos/test-org/locked/releases/tags/v1.2.3"] + calls["GET https://api.github.com/repos/Test-Org/Locked/releases/tags/v1.2.3"]; n != 1 {
		t.Errorf("test-org/locked@v1.2.3 looked up %d times, want 1", n)
	}
	if n := calls["GET https://api.github.com/repos/test-org/norelease/releases/tags/v1"]; n != 1 {
		t.Errorf("test-org/norelease@v1 looked up %d times, want 1 (a missing release is an answer)", n)
	}
	if n := calls["GET https://api.github.com/repos/test-org/broken/releases/tags/v1.0.0"]; n != 2 {
		t.Errorf("test-org/broken@v1.0.0 looked up %d times, want 2 (a failed lookup is retried)", n)
	}
}
