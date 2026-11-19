package hardenrunner

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v40/github"
	"github.com/jarcoal/httpmock"
)

// When the harden-runner step is added but its commit lookup fails, AddAction
// must return the output with the step at its tag together with GitHub's own
// error, unchanged. Discarding that error let a rate limit ship an unpinned
// Harden Runner as a complete result.
func TestAddActionReturnsPinErrorWithUnpinnedStep(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/commits/v2",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(http.StatusForbidden, `{"message":"API rate limit exceeded for installation ID 1234."}`)
			resp.Header.Set("Content-Type", "application/json")
			resp.Header.Set("X-RateLimit-Remaining", "0")
			resp.Request = req
			return resp, nil
		})

	input := `name: ci
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`
	out, updated, err := AddAction(input, HardenRunnerConfig{}, true, false, false)

	var rateLimited *github.RateLimitError
	if !errors.As(err, &rateLimited) {
		t.Fatalf("AddAction() error = %v (%T), want go-github's *github.RateLimitError returned unchanged", err, err)
	}
	if !updated {
		t.Error("updated = false, want true: the step was added even though it could not be pinned")
	}
	if !strings.Contains(out, "uses: step-security/harden-runner@v2") {
		t.Errorf("the step must be kept at its tag when pinning fails:\n%s", out)
	}
}
