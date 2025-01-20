package pin

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type customTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Host, "ghcr.io") {
		req2 := req.Clone(req.Context())
		req2.URL.Scheme = "https"
		req2.URL.Host = strings.TrimPrefix(t.baseURL, "https://")
		return t.base.RoundTrip(req2)
	}
	return t.base.RoundTrip(req)
}

func createGhesTestServer(t *testing.T) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		if !strings.Contains(r.Host, "ghcr.io") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Mock manifest endpoints
		switch r.URL.Path {

		case "/v2/": // simulate ping request
			w.WriteHeader(http.StatusOK)

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
		case "/v2/actions/checkout/manifests/1.2.0":
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
	server := createGhesTestServer(t)
	defer server.Close()

	// Create a custom client that redirects ghcr.io to our test server
	originalClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: &customTransport{
			base: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			baseURL: server.URL,
		},
	}

	// update default transport
	OriginalTransport := http.DefaultTransport
	http.DefaultTransport = http.DefaultClient.Transport

	defer func() {
		http.DefaultClient = originalClient
		http.DefaultTransport = OriginalTransport
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
	actionPath = strings.TrimPrefix(actionPath, "/v2/")

	fileName := strings.ReplaceAll(actionPath, "/", "-") + ".json"
	testFilesDir := "../../../testfiles/pinactions/immutableActionResponses/"
	respFilePath := filepath.Join(testFilesDir, fileName)

	resp, err := ioutil.ReadFile(respFilePath)
	if err != nil {
		t.Fatalf("error reading test file:%v", err)
	}

	return resp
}
