package workflow

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v40/github"
	"github.com/jarcoal/httpmock"
)

// When the Harden Runner step cannot be pinned, SecureWorkflow must return
// GitHub's error instead of reporting success. It used to log the failure and
// return nil, so a rate limit on that one lookup produced a workflow the caller
// shipped as complete, with Harden Runner unpinned.
func TestSecureWorkflowReturnsHardenRunnerPinError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/harden-runner/commits/v2",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(http.StatusForbidden, `{"message":"API rate limit exceeded for installation ID 1234."}`)
			resp.Header.Set("Content-Type", "application/json")
			resp.Header.Set("X-RateLimit-Remaining", "0")
			// A real response always carries its request; go-github's rate-limit
			// error formats the method and URL from it.
			resp.Request = req
			return resp, nil
		})

	input := `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: make test
`
	params := map[string]string{
		"pinActions":       "true",
		"addHardenRunner":  "true",
		"addPermissions":   "false",
		"ignoreMissingKBs": "true",
	}

	resp, err := SecureWorkflow(params, input, nil)

	var rateLimited *github.RateLimitError
	if !errors.As(err, &rateLimited) {
		t.Fatalf("SecureWorkflow() error = %v (%T), want go-github's *github.RateLimitError returned unchanged", err, err)
	}
	if resp == nil || !resp.HasErrors {
		t.Fatalf("response = %+v, want a response with HasErrors set", resp)
	}
	if strings.Contains(resp.FinalOutput, "step-security/harden-runner") {
		t.Errorf("the unpinned Harden Runner step must not be in the output of a failed run:\n%s", resp.FinalOutput)
	}
	if strings.TrimSpace(resp.FinalOutput) == "" {
		t.Error("FinalOutput must never be empty for a non-empty input")
	}
}
