package workflow

import (
	"context"
	"encoding/base64"
	"os"

	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
)

func GetGitHubWorkflowContents(queryStringParams map[string]string) (string, error) {

	PAT := os.Getenv("PAT")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: PAT},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	owner := queryStringParams["owner"]
	repo := queryStringParams["repo"]
	path := queryStringParams["path"]
	branch := queryStringParams["branch"]

	content, _, _, err := client.Repositories.GetContents(ctx,
		owner,
		repo,
		path,
		&github.RepositoryContentGetOptions{Ref: branch})

	if err != nil {
		return "", err
	}

	workflowYaml, err := base64.StdEncoding.DecodeString(*content.Content)

	if err != nil {
		return "", err
	}

	return string(workflowYaml), nil
}
