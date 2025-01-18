package pin

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

var (
	testFilesDir = "../../../testfiles/pinactions/immutableActionResponses/"
)

func createTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		// Mock manifest endpoints
		switch r.URL.Path {

		case "/token":
			// for immutable actions, since image will be present in registry...it returns 200 OK with token
			// otherwise it returns 403 Forbidden
			scope := r.URL.Query().Get("scope")
			switch scope {
			case "repository:actions/checkout:pull":
				fallthrough
			case "repository:step-security/wait-for-secrets:pull":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"token": "test-token", "access_token": "test-token"}`))
			default:
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"errors": [{"code": "DENIED", "message": "requested access to the resource is denied"}]}`))
			}

		case "/v2/actions/checkout/manifests/4.2.2":
			fallthrough
		case "/v2/step-security/wait-for-secrets/manifests/1.2.0":
			w.Write(readHttpResponseForAction(t, r.URL.Path))
		case "/v2/actions/checkout/manifests/1.2.3": // since this version doesn't exist
			fallthrough
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write(readHttpResponseForAction(t, "default"))
		}
	}))
}

func Test_isImmutableAction(t *testing.T) {
	// Create test server that mocks GitHub Container Registry
	server := createTestServer(t)
	defer server.Close()

	// Create a custom client that redirects ghcr.io to our test server
	originalClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				if strings.Contains(req.URL.Host, "ghcr.io") {
					return url.Parse(server.URL)
				}
				return nil, nil
			},
		},
	}
	defer func() {
		http.DefaultClient = originalClient
	}()

	tests := []struct {
		name   string
		action string
		want   bool
	}{
		{
			name:   "immutable action - 1",
			action: "actions/checkout@v4.2.2",
			want:   true,
		},
		{
			name:   "immutable action - 2",
			action: "step-security/wait-for-secrets@v1.2.0",
			want:   true,
		},
		{
			name:   "non immutable action(valid action)",
			action: "sailikhith-stepsecurity/hello-action@v1.0.2",
			want:   false,
		},
		{
			name:   "non immutable action(invalid action)",
			action: "sailikhith-stepsecurity/no-such-action@v1.0.2",
			want:   false,
		},
		{
			name:   " action with release tag doesn't exist",
			action: "actions/checkout@1.2.3",
			want:   false,
		},
		{
			name:   "invalid action format",
			action: "invalid-format",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := IsImmutableAction(tt.action)
			if got != tt.want {
				t.Errorf("isImmutableAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func readHttpResponseForAction(t *testing.T, actionPath string) []byte {
	// remove v2 prefix from action path
	actionPath = strings.TrimPrefix(actionPath, "v2/")

	fileName := strings.ReplaceAll(actionPath, "/", "-")
	respFilePath := testFilesDir + fileName

	resp, err := ioutil.ReadFile(respFilePath)
	if err != nil {
		t.Fatalf("error reading test file")
	}

	return resp
}
