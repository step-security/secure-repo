package pin

import (
	"net/http"
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
