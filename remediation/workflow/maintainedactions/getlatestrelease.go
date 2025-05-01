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
	if len(splitOnSlash) != 2 {
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
