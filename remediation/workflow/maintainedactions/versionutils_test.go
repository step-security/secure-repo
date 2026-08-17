package maintainedactions

import (
	"io/ioutil"
	"net/http"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestGetMajorVersion(t *testing.T) {
	cases := map[string]string{
		"v5":     "v5",
		"v5.5.5": "v5",
		"5.5.5":  "5",
		"5":      "5",
		"v":      "v",
		"":       "",
	}
	for in, want := range cases {
		if got := getMajorVersion(in); got != want {
			t.Errorf("getMajorVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGetLatestRelease_InvalidRepo(t *testing.T) {
	if _, err := GetLatestRelease("no-slash"); err == nil {
		t.Fatal("expected error for invalid owner/repo")
	}
}

func TestGetLatestRelease_NoPATFails(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Setenv("PAT", "")
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/releases/latest",
		httpmock.NewStringResponder(500, `{"message":"boom"}`))
	if _, err := GetLatestRelease("owner/repo"); err == nil {
		t.Fatal("expected error when first call fails and no PAT is set")
	}
}

func TestGetLatestRelease_PATRetrySucceeds(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Setenv("PAT", "fake-token")
	calls := 0
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/releases/latest",
		func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return httpmock.NewStringResponse(500, `{"message":"boom"}`), nil
			}
			return httpmock.NewStringResponse(200, `{"tag_name":"v3.2.1"}`), nil
		})
	v, err := GetLatestRelease("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v3" {
		t.Errorf("got %q, want v3", v)
	}
}

func TestGetLatestRelease_PATRetryFails(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Setenv("PAT", "fake-token")
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/releases/latest",
		httpmock.NewStringResponder(500, `{"message":"boom"}`))
	if _, err := GetLatestRelease("owner/repo"); err == nil {
		t.Fatal("expected error when both attempts fail")
	}
}

// GetTagFromSHA

func TestGetTagFromSHA_InvalidRepo(t *testing.T) {
	if _, err := GetTagFromSHA("no-slash", "abc"); err == nil {
		t.Fatal("expected error for invalid owner/repo")
	}
}

func TestGetTagFromSHA_ListError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(500, `{"message":"boom"}`))
	if _, err := GetTagFromSHA("owner/repo", "anything"); err == nil {
		t.Fatal("expected error from ListMatchingRefs failure")
	}
}

func TestGetTagFromSHA_CommitMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v2.0.0","object":{"sha":"aaaa","type":"commit"}},
			{"ref":"refs/tags/v5.1.0","object":{"sha":"bbbb","type":"commit"}}
		]`))
	v, err := GetTagFromSHA("owner/repo", "bbbb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v5.1.0" {
		t.Errorf("got %q, want v5.1.0 (full tag)", v)
	}
}

func TestGetTagFromSHA_PrefersConcreteOverBareMajor(t *testing.T) {
	// A release commit is tagged with both "v4" and "v4.2.1"; the concrete
	// version is what the workflow is effectively pinned to.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v4","object":{"sha":"same","type":"commit"}},
			{"ref":"refs/tags/v4.2.1","object":{"sha":"same","type":"commit"}}
		]`))
	v, err := GetTagFromSHA("owner/repo", "same")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v4.2.1" {
		t.Errorf("got %q, want v4.2.1", v)
	}
}

func TestGetTagFromSHA_BareMajorFallback(t *testing.T) {
	// Only a bare major points at the commit -> returned as the fallback.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v4","object":{"sha":"same","type":"commit"}}
		]`))
	v, err := GetTagFromSHA("owner/repo", "same")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v4" {
		t.Errorf("got %q, want v4 (bare-major fallback)", v)
	}
}

func TestGetTagFromSHA_AnnotatedTagMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v3.0.0","object":{"sha":"tagsha","type":"tag"}}
		]`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/commits/refs/tags/v3.0.0",
		httpmock.NewStringResponder(200, `commitsha`))
	v, err := GetTagFromSHA("owner/repo", "commitsha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v3.0.0" {
		t.Errorf("got %q, want v3.0.0", v)
	}
}

func TestGetTagFromSHA_AnnotatedDerefError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v3.0.0","object":{"sha":"tagsha","type":"tag"}},
			{"ref":"refs/tags/v4.0.0","object":{"sha":"match","type":"commit"}}
		]`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/commits/refs/tags/v3.0.0",
		httpmock.NewStringResponder(500, `{"message":"boom"}`))
	v, err := GetTagFromSHA("owner/repo", "match")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v4.0.0" {
		t.Errorf("got %q, want v4.0.0 (deref error should be skipped, not fatal)", v)
	}
}

func TestGetTagFromSHA_NoMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v2.0.0","object":{"sha":"aaaa","type":"commit"}}
		]`))
	v, err := GetTagFromSHA("owner/repo", "nomatch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "" {
		t.Errorf("got %q, want empty string", v)
	}
}

func TestGetTagFromSHA_WithPAT(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Setenv("PAT", "fake-token")
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v1.0.0","object":{"sha":"match","type":"commit"}}
		]`))
	v, err := GetTagFromSHA("owner/repo", "match")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v1.0.0" {
		t.Errorf("got %q, want v1.0.0", v)
	}
}

// GetMajorTagIfExists

func TestGetMajorTagIfExists_InvalidRepo(t *testing.T) {
	if _, _, err := GetMajorTagIfExists("no-slash", "v1"); err == nil {
		t.Fatal("expected error for invalid owner/repo")
	}
}

func TestGetMajorTagIfExists_NonErrorNoPAT(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Setenv("PAT", "")
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/ref/tags/v5",
		httpmock.NewStringResponder(500, `{"message":"boom"}`))
	tag, exists, err := GetMajorTagIfExists("owner/repo", "v5")
	if err != nil || exists || tag != "" {
		t.Errorf("got tag=%q exists=%v err=%v, want empty/false/nil", tag, exists, err)
	}
}

func TestGetMajorTagIfExists_PATRetrySucceeds(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Setenv("PAT", "fake-token")
	calls := 0
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/ref/tags/v5",
		func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return httpmock.NewStringResponse(500, `{"message":"boom"}`), nil
			}
			return httpmock.NewStringResponse(200,
				`{"ref":"refs/tags/v5","object":{"sha":"x","type":"commit"}}`), nil
		})
	tag, exists, err := GetMajorTagIfExists("owner/repo", "v5")
	if err != nil || !exists || tag != "v5" {
		t.Errorf("got tag=%q exists=%v err=%v, want v5/true/nil", tag, exists, err)
	}
}

func TestGetMajorTagIfExists_PATRetry404(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Setenv("PAT", "fake-token")
	calls := 0
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/ref/tags/v5",
		func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return httpmock.NewStringResponse(500, `{"message":"boom"}`), nil
			}
			return httpmock.NewStringResponse(404, `{"message":"Not Found"}`), nil
		})
	tag, exists, err := GetMajorTagIfExists("owner/repo", "v5")
	if err != nil || exists || tag != "" {
		t.Errorf("got tag=%q exists=%v err=%v, want empty/false/nil", tag, exists, err)
	}
}

func TestGetMajorTagIfExists_PATRetryFails(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Setenv("PAT", "fake-token")
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/owner/repo/git/ref/tags/v5",
		httpmock.NewStringResponder(500, `{"message":"boom"}`))
	_, exists, err := GetMajorTagIfExists("owner/repo", "v5")
	if err == nil {
		t.Fatal("expected wrapped error when both attempts fail with non-404")
	}
	if exists {
		t.Error("expected exists=false")
	}
}

// resolveVersion

func TestResolveVersion_NoRef(t *testing.T) {
	if _, err := resolveVersion("actions/checkout", "actions/checkout", "new/action", true); err == nil {
		t.Fatal("expected error when originalUses has no @ref")
	}
}

func TestResolveVersion_SHAResolved(t *testing.T) {
	// SHA resolves to v5.2.0 and the fork has that exact version -> replace with v5.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/orig/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v5.2.0","object":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","type":"commit"}}
		]`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v5",
		httpmock.NewStringResponder(200,
			`{"ref":"refs/tags/v5","object":{"sha":"x","type":"commit"}}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v5.2.0",
		httpmock.NewStringResponder(200,
			`{"ref":"refs/tags/v5.2.0","object":{"sha":"y","type":"commit"}}`))
	uses := "orig/repo@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	v, err := resolveVersion(uses, "orig/repo", "new/repo", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v5" {
		t.Errorf("got %q, want v5", v)
	}
}

func TestResolveVersion_MajorTagResolvedThroughSHA(t *testing.T) {
	// The workflow pins "@v4", which currently points at v4.2.1. The fork has v4
	// but has not released v4.2.1, so replacing would move the workflow back.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v4",
		httpmock.NewStringResponder(200, `{"ref":"refs/tags/v4","object":{"sha":"x","type":"commit"}}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/orig/repo/commits/refs/tags/v4",
		httpmock.NewStringResponder(200, `origsha`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/orig/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[
			{"ref":"refs/tags/v4","object":{"sha":"origsha","type":"commit"}},
			{"ref":"refs/tags/v4.2.1","object":{"sha":"origsha","type":"commit"}}
		]`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v4.2.1",
		httpmock.NewStringResponder(404, `{"message":"Not Found"}`))
	if _, err := resolveVersion("orig/repo@v4", "orig/repo", "new/repo", true); err == nil {
		t.Fatal("expected skip when the fork lacks the version the major tag points at")
	}
}

func TestResolveVersion_ForkMissingVersionSkips(t *testing.T) {
	// Concrete "@v4.2.1" that the fork has not released -> skip.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v4",
		httpmock.NewStringResponder(200, `{"ref":"refs/tags/v4","object":{"sha":"x","type":"commit"}}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v4.2.1",
		httpmock.NewStringResponder(404, `{"message":"Not Found"}`))
	if _, err := resolveVersion("orig/repo@v4.2.1", "orig/repo", "new/repo", true); err == nil {
		t.Fatal("expected skip when fork lacks the version")
	}
}

func TestResolveVersion_ForkHasVersionReplaces(t *testing.T) {
	// Concrete "@v4.2.1" that the fork does have -> replace with the major tag.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v4",
		httpmock.NewStringResponder(200, `{"ref":"refs/tags/v4","object":{"sha":"x","type":"commit"}}`))
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v4.2.1",
		httpmock.NewStringResponder(200, `{"ref":"refs/tags/v4.2.1","object":{"sha":"y","type":"commit"}}`))
	v, err := resolveVersion("orig/repo@v4.2.1", "orig/repo", "new/repo", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v4" {
		t.Errorf("got %q, want v4", v)
	}
}

func TestResolveVersion_NoMajorTagSkipsVersionLookup(t *testing.T) {
	// The fork has no matching major tag, so nothing further is looked up: only
	// the major-tag check is mocked, and any extra call would fail the test.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/new/repo/git/ref/tags/v3",
		httpmock.NewStringResponder(404, `{"message":"Not Found"}`))
	if _, err := resolveVersion("orig/repo@v3.1.0", "orig/repo", "new/repo", true); err == nil {
		t.Fatal("expected error when fork has no matching major tag")
	}
	if n := httpmock.GetTotalCallCount(); n != 1 {
		t.Errorf("made %d API calls, want 1 (no version resolution once the major is missing)", n)
	}
}

func TestResolveVersion_SHALookupFails(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/orig/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(500, `{"message":"boom"}`))
	uses := "orig/repo@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := resolveVersion(uses, "orig/repo", "new/repo", true); err == nil {
		t.Fatal("expected error when SHA lookup fails")
	}
}

func TestResolveVersion_SHANoMatch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/orig/repo/git/matching-refs/tags/v",
		httpmock.NewStringResponder(200, `[]`))
	uses := "orig/repo@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := resolveVersion(uses, "orig/repo", "new/repo", true); err == nil {
		t.Fatal("expected error when SHA has no matching tag")
	}
}

// LoadMaintainedActions

func TestLoadMaintainedActions_ReadError(t *testing.T) {
	if _, err := LoadMaintainedActions("/nonexistent/path/to/file.json"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadMaintainedActions_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	f := path.Join(dir, "bad.json")
	if err := ioutil.WriteFile(f, []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := LoadMaintainedActions(f); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// ReplaceActions

func TestReplaceActions_InvalidYAML(t *testing.T) {
	// A mapping key cannot also be a sequence at the same indent — yaml.Unmarshal errors.
	bad := "foo: bar\n- item"
	if _, _, err := ReplaceActions(bad, map[string]string{}, false); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestReplaceActions_ReusableWorkflowSkipped(t *testing.T) {
	// A job that calls a reusable workflow (has top-level `uses:`) should be skipped.
	// No HTTP calls should be made, and no replacements should occur.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	input := `name: reusable
on: push
jobs:
  call:
    uses: ./.github/workflows/other.yml
`
	actionMap := map[string]string{"amannn/action-semantic-pull-request": "step-security/action-semantic-pull-request"}
	got, updated, err := ReplaceActions(input, actionMap, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Error("expected updated=false for reusable workflow")
	}
	if got != input {
		t.Errorf("expected input unchanged, got %q", got)
	}
}

func TestReplaceActions_CompositeResolverFailureSkipped(t *testing.T) {
	// Composite action where the resolver fails on the single mapped step should
	// skip that step (log + continue) and return unchanged input.
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	// Fork has no matching major version.
	httpmock.RegisterResponder("GET",
		"https://api.github.com/repos/step-security/action-semantic-pull-request/git/ref/tags/v3",
		httpmock.NewStringResponder(404, `{"message":"Not Found"}`))

	input := `name: composite
runs:
  using: composite
  steps:
    - uses: amannn/action-semantic-pull-request@v3
`
	actionMap := map[string]string{
		"amannn/action-semantic-pull-request": "step-security/action-semantic-pull-request",
	}
	got, updated, err := ReplaceActions(input, actionMap, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Error("expected updated=false when composite resolver fails")
	}
	if got != input {
		t.Errorf("expected input unchanged, got %q", got)
	}
}
