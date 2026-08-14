package maintainedactions

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
)

type Release struct {
	TagName string `json:"tag_name"`
}

// isConcreteSemver reports whether v pins at least a minor version
// (e.g. "v4.2" or "v4.2.1"), as opposed to a bare major ("v4") or empty.
// Only concrete versions can be meaningfully compared for downgrades.
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

// GetMajorTagFromSHA finds the major version tag (e.g., "v5") on ownerRepo
// whose commit matches the given SHA, by listing all tags with prefix "tags/v".
// Returns ("", nil) if no matching tag is found.
func GetMajorTagFromSHA(ownerRepo, sha string) (string, error) {
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
		if refSHA == sha {
			tag := strings.TrimPrefix(ref.GetRef(), "refs/tags/")
			return getMajorVersion(tag), nil
		}
	}
	return "", nil
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

// GetLatestTagForMajor returns the highest concrete semantic-version tag on
// ownerRepo for the given majorVersion (e.g. majorVersion "v4" -> "v4.3.1").
// It lists tags with the prefix "<majorVersion>." and picks the largest by
// semver. Returns ("", nil) when no concrete tag exists for that major.
func GetLatestTagForMajor(ownerRepo, majorVersion string) (string, error) {
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

	refs, _, err := client.Git.ListMatchingRefs(ctx, owner, repo, &github.ReferenceListOptions{
		Ref: "tags/" + majorVersion + ".",
	})
	if err != nil {
		return "", err
	}

	best := ""
	for _, ref := range refs {
		tag := strings.TrimPrefix(ref.GetRef(), "refs/tags/")
		if !isConcreteSemver(tag) {
			continue
		}
		if best == "" || compareSemver(tag, best) > 0 {
			best = tag
		}
	}
	return best, nil
}
