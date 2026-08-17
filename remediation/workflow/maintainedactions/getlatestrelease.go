package maintainedactions

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
)

type Release struct {
	TagName string `json:"tag_name"`
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
