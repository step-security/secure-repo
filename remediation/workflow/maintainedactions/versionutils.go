package maintainedactions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
)

// isRefNotFound reports whether err means "this ref does not exist", so callers
// can tell that apart from a transport or rate-limit failure. Looking up a
// missing ref answers 404 on most endpoints, but /commits/{ref} answers
// 422 ("No commit found for SHA: refs/tags/v3"), so both count.
func isRefNotFound(err error) bool {
	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		switch errResp.Response.StatusCode {
		case http.StatusNotFound, http.StatusUnprocessableEntity:
			return true
		}
	}
	return false
}

type Release struct {
	TagName string `json:"tag_name"`
}

// isConcreteSemver reports whether v pins at least a minor version
// (e.g. "v4.2" or "v4.2.1"), as opposed to a bare major ("v4") or empty.
func isConcreteSemver(v string) bool {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return false
	}
	return strings.Count(v, ".") >= 1
}

// parseSemverParts extracts [major, minor, patch] from a version string like
// "v4.2.1". Missing components default to 0; prerelease/build metadata and any
// non-numeric component are ignored (treated as 0).
func parseSemverParts(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out [3]int
	for i, p := range strings.Split(v, ".") {
		if i > 2 {
			break
		}
		n, _ := strconv.Atoi(p)
		out[i] = n
	}
	return out
}

// compareSemver returns -1 if a < b, 0 if equal, 1 if a > b, comparing only the
// numeric major.minor.patch components.
func compareSemver(a, b string) int {
	pa, pb := parseSemverParts(a), parseSemverParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

// compareMajorMinor is like compareSemver but ignores the patch component, so a
// patch-level difference (e.g. v4.2.3 vs v4.2.1) compares equal while a minor or
// major difference still orders. Used where a patch downgrade is acceptable but a
// minor/major downgrade is not.
func compareMajorMinor(a, b string) int {
	pa, pb := parseSemverParts(a), parseSemverParts(b)
	for i := 0; i < 2; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func getMajorVersion(version string) string {
	hasVPrefix := strings.HasPrefix(version, "v")
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		if hasVPrefix {
			return "v" + parts[0]
		}
		return parts[0]
	}
	if hasVPrefix {
		return "v" + version
	}
	return version
}

func GetLatestRelease(ownerRepo string) (string, error) {
	splitOnSlash := strings.Split(ownerRepo, "/")
	if len(splitOnSlash) < 2 {
		return "", fmt.Errorf("invalid owner/repo format: %s", ownerRepo)
	}
	owner := splitOnSlash[0]
	repo := splitOnSlash[1]

	ctx := context.Background()

	// First try without token
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		// If failed, try with token
		token := os.Getenv("PAT")
		if token == "" {
			return "", fmt.Errorf("failed to get latest release and no GITHUB_TOKEN available: %w", err)
		}

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)

		release, _, err = client.Repositories.GetLatestRelease(ctx, owner, repo)
		if err != nil {
			return "", fmt.Errorf("failed to get latest release with token: %w", err)
		}
	}

	return getMajorVersion(release.GetTagName()), nil
}

// GetTagFromSHA finds the version tag on ownerRepo whose commit matches the
// given SHA, by listing all tags with prefix "tags/v". A release commit is
// normally tagged both with the concrete version ("v5.2.0") and the floating
// major ("v5"); the concrete one is returned, since that is the version the
// workflow is effectively pinned to. Falls back to the bare major when no
// concrete tag points at the commit.
// Returns ("", nil) if no matching tag is found.
func GetTagFromSHA(ownerRepo, sha string) (string, error) {
	splitOnSlash := strings.Split(ownerRepo, "/")
	if len(splitOnSlash) < 2 {
		return "", fmt.Errorf("invalid owner/repo format: %s", ownerRepo)
	}
	owner := splitOnSlash[0]
	repo := splitOnSlash[1]

	ctx := context.Background()
	client := github.NewClient(nil)

	token := os.Getenv("PAT")
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		client = github.NewClient(oauth2.NewClient(ctx, ts))
	}

	refs, _, err := client.Git.ListMatchingRefs(ctx, owner, repo, &github.ReferenceListOptions{
		Ref: "tags/v",
	})
	if err != nil {
		return "", err
	}

	fallback := ""
	for _, ref := range refs {
		var refSHA string
		if ref.GetObject().GetType() == "commit" {
			refSHA = ref.GetObject().GetSHA()
		} else {
			// annotated tag — dereference to get the commit SHA
			refSHA, _, err = client.Repositories.GetCommitSHA1(ctx, owner, repo, ref.GetRef(), "")
			if err != nil {
				continue
			}
		}
		if refSHA != sha {
			continue
		}
		tag := strings.TrimPrefix(ref.GetRef(), "refs/tags/")
		if isConcreteSemver(tag) {
			return tag, nil
		}
		if fallback == "" {
			fallback = tag
		}
	}
	return fallback, nil
}

// tagForSHA resolves a commit SHA on ownerRepo to the version tag on it,
// treating "no tag found" as an error since the version cannot be determined.
func tagForSHA(ownerRepo, sha string) (string, error) {
	version, err := GetTagFromSHA(ownerRepo, sha)
	if err != nil {
		return "", fmt.Errorf("unable to resolve SHA %s to a tag: %w", sha, err)
	}
	if version == "" {
		return "", fmt.Errorf("unable to resolve SHA %s to a tag", sha)
	}
	return version, nil
}

// GetSHAFromTag resolves a tag (e.g. "v5" or "v5.2.0") on ownerRepo to the
// commit SHA it points at, dereferencing annotated tags.
func GetSHAFromTag(ownerRepo, tag string) (string, error) {
	splitOnSlash := strings.Split(ownerRepo, "/")
	if len(splitOnSlash) < 2 {
		return "", fmt.Errorf("invalid owner/repo format: %s", ownerRepo)
	}
	owner := splitOnSlash[0]
	repo := splitOnSlash[1]

	ctx := context.Background()
	client := github.NewClient(nil)
	if token := os.Getenv("PAT"); token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		client = github.NewClient(oauth2.NewClient(ctx, ts))
	}

	sha, _, err := client.Repositories.GetCommitSHA1(ctx, owner, repo, "refs/tags/"+tag, "")
	if err != nil {
		return "", err
	}
	return sha, nil
}

// GetSHAFromBranch resolves a branch (e.g. "v3") on ownerRepo to the commit SHA
// at its head. Most actions publish their floating major version as a tag, but
// some use a branch instead — arduino/setup-task and
// JarvusInnovations/background-action both do — so this is the fallback when the
// tag lookup comes back 404.
func GetSHAFromBranch(ownerRepo, branch string) (string, error) {
	splitOnSlash := strings.Split(ownerRepo, "/")
	if len(splitOnSlash) < 2 {
		return "", fmt.Errorf("invalid owner/repo format: %s", ownerRepo)
	}
	owner := splitOnSlash[0]
	repo := splitOnSlash[1]

	ctx := context.Background()
	client := github.NewClient(nil)
	if token := os.Getenv("PAT"); token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		client = github.NewClient(oauth2.NewClient(ctx, ts))
	}

	sha, _, err := client.Repositories.GetCommitSHA1(ctx, owner, repo, "refs/heads/"+branch, "")
	if err != nil {
		return "", err
	}
	return sha, nil
}

// VersionForMajorTag returns the concrete semantic version that a major tag on
// ownerRepo currently points at (e.g. "v0" -> "v0.6.2"), by resolving the tag to
// its commit and reading the concrete tag on that commit. This is the version a
// workflow using the major tag actually runs.
func VersionForMajorTag(ownerRepo, majorTag string) (string, error) {
	sha, err := GetSHAFromTag(ownerRepo, majorTag)
	if err != nil {
		return "", fmt.Errorf("unable to resolve tag %s on %s to a commit SHA: %w", majorTag, ownerRepo, err)
	}
	return tagForSHA(ownerRepo, sha)
}

// GetMajorTagIfExists checks whether ownerRepo has a tag exactly matching
// majorVersion (e.g., "v5"). Returns (majorVersion, true, nil) when the tag
// exists, ("", false, nil) when it is absent (404), and ("", false, err) for
// unexpected API failures.
func GetMajorTagIfExists(ownerRepo, majorVersion string) (string, bool, error) {
	splitOnSlash := strings.Split(ownerRepo, "/")
	if len(splitOnSlash) < 2 {
		return "", false, fmt.Errorf("invalid owner/repo format: %s", ownerRepo)
	}
	owner := splitOnSlash[0]
	repo := splitOnSlash[1]

	ctx := context.Background()
	client := github.NewClient(nil)

	_, resp, err := client.Git.GetRef(ctx, owner, repo, "refs/tags/"+majorVersion)
	if err == nil {
		return majorVersion, true, nil
	}
	if resp != nil && resp.StatusCode == 404 {
		return "", false, nil
	}

	// First attempt failed for a non-404 reason — retry with token.
	token := os.Getenv("PAT")
	if token == "" {
		return "", false, nil
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client = github.NewClient(tc)

	_, resp, err = client.Git.GetRef(ctx, owner, repo, "refs/tags/"+majorVersion)
	if err == nil {
		return majorVersion, true, nil
	}
	if resp != nil && resp.StatusCode == 404 {
		return "", false, nil
	}
	return "", false, fmt.Errorf("failed to check tag %s on %s: %w", majorVersion, ownerRepo, err)
}
