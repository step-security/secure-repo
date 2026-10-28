package pin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Option configures PinActions, PinActionWithPatFallback and PinAction.
type Option func(*options)

type options struct {
	skipImmutableReleaseTags bool
}

// WithSkipImmutableReleaseTags leaves an action reference unpinned when its tag
// belongs to an immutable GitHub release, whose tag cannot be moved or deleted.
func WithSkipImmutableReleaseTags(skip bool) Option {
	return func(o *options) {
		o.skipImmutableReleaseTags = skip
	}
}

func applyOptions(opts []Option) options {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// githubAPIBaseURL is the REST API the release lookups go to.
const githubAPIBaseURL = "https://api.github.com"

// IsImmutableReleaseAction reports whether an action reference
// ("owner/repo[/path]@tag") uses a tag that belongs to an immutable GitHub
// release. References that cannot be one (local actions, docker images, no ref,
// or already pinned to a commit SHA) return false. It reads the release with the
// token pinning uses (SECURE_REPO_PAT, else PAT).
func IsImmutableReleaseAction(action string) (bool, error) {
	action = strings.TrimSpace(action)
	if !strings.Contains(action, "@") || strings.HasPrefix(action, "./") || strings.HasPrefix(action, "docker://") || isAbsolute(action) {
		return false, nil
	}
	name, tag, _ := strings.Cut(action, "@")
	parts := strings.Split(name, "/")
	if tag == "" || len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return false, nil
	}
	token := os.Getenv("SECURE_REPO_PAT")
	if token == "" {
		token = os.Getenv("PAT")
	}
	return isImmutableReleaseTag(token, parts[0], parts[1], tag)
}

// isImmutableReleaseTag reads GET /repos/{owner}/{repo}/releases/tags/{tag} and
// reports the release's "immutable" field. A tag with no release (404) is not
// immutable. The request is made directly rather than with go-github, whose
// v40 release type predates the field.
func isImmutableReleaseTag(token, owner, repo, tag string) (bool, error) {
	key := owner + "/" + repo + "@" + tag

	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", githubAPIBaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(tag))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("get release %s: %w", key, err)
	}
	defer resp.Body.Close()

	var release struct {
		Immutable bool `json:"immutable"`
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
	case resp.StatusCode != http.StatusOK:
		return false, fmt.Errorf("get release %s: status %d", key, resp.StatusCode)
	default:
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return false, fmt.Errorf("decode release %s: %w", key, err)
		}
	}
	return release.Immutable, nil
}
