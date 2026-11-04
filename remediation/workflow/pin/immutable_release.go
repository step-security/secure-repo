package pin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
)

// Option configures PinActions, PinActionWithPatFallback and PinAction.
type Option func(*options)

type options struct {
	skipImmutableReleaseTags bool
	immutableReleaseCache    *ImmutableReleaseCache
}

// WithSkipImmutableReleaseTags leaves an action reference unpinned when its tag
// belongs to an immutable GitHub release, whose tag cannot be moved or deleted.
// This includes references listed in actionCommitMap.
func WithSkipImmutableReleaseTags(skip bool) Option {
	return func(o *options) {
		o.skipImmutableReleaseTags = skip
	}
}

// WithImmutableReleaseCache shares immutable-release lookups through cache, so
// each owner/repo@tag is looked up once however often it is referenced. Without
// it every reference is looked up.
func WithImmutableReleaseCache(cache *ImmutableReleaseCache) Option {
	return func(o *options) {
		o.immutableReleaseCache = cache
	}
}

// ImmutableReleaseCache remembers whether a tag belongs to an immutable GitHub
// release, keyed by owner/repo@tag, so a sub-path action ("owner/repo/sub@tag")
// shares its repository's entry. It is meant to live for one remediation run,
// created by the caller and passed to every pinning call in that run: a tag that
// has no release now can get one later, so results are not kept across runs.
// Each owner/repo@tag is looked up once per run, also when several goroutines
// ask for it at the same time: concurrent callers wait for the lookup already
// in flight instead of starting their own. Only answers are kept; a failed
// lookup is returned to the callers waiting on it and retried by the next one.
// Safe for concurrent use.
type ImmutableReleaseCache struct {
	results sync.Map
	// inFlight coalesces concurrent lookups of the same key into one request.
	inFlight singleflight.Group
}

// NewImmutableReleaseCache returns an empty cache for one remediation run.
func NewImmutableReleaseCache() *ImmutableReleaseCache {
	return &ImmutableReleaseCache{}
}

func immutableReleaseCacheKey(owner, repo, tag string) string {
	// GitHub owner and repository names are case-insensitive; tags are not.
	return strings.ToLower(owner) + "/" + strings.ToLower(repo) + "@" + tag
}

// lookupImmutableRelease reads a tag's release through cache when there is one.
func lookupImmutableRelease(cache *ImmutableReleaseCache, token, owner, repo, tag string) (bool, error) {
	if cache == nil {
		return isImmutableReleaseTag(token, owner, repo, tag)
	}
	key := immutableReleaseCacheKey(owner, repo, tag)
	if immutable, ok := cache.results.Load(key); ok {
		return immutable.(bool), nil
	}
	result, err, _ := cache.inFlight.Do(key, func() (interface{}, error) {
		// Another caller may have stored the answer between the Load above and
		// this lookup becoming the one in flight.
		if immutable, ok := cache.results.Load(key); ok {
			return immutable, nil
		}
		immutable, err := isImmutableReleaseTag(token, owner, repo, tag)
		if err != nil {
			return false, err
		}
		cache.results.Store(key, immutable)
		return immutable, nil
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
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
// token pinning uses (SECURE_REPO_PAT, else PAT). Pass WithImmutableReleaseCache
// to share lookups across calls.
func IsImmutableReleaseAction(action string, opts ...Option) (bool, error) {
	action = strings.TrimSpace(action)
	if !strings.Contains(action, "@") || strings.HasPrefix(action, "./") || strings.HasPrefix(action, "docker://") || isAbsolute(action) {
		return false, nil
	}
	// strings.SplitN rather than strings.Cut: this module builds with Go 1.17.
	atParts := strings.SplitN(action, "@", 2)
	name, tag := atParts[0], atParts[1]
	parts := strings.Split(name, "/")
	if tag == "" || len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return false, nil
	}
	token := os.Getenv("SECURE_REPO_PAT")
	if token == "" {
		token = os.Getenv("PAT")
	}
	return lookupImmutableRelease(applyOptions(opts).immutableReleaseCache, token, parts[0], parts[1], tag)
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
