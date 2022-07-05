package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

func PinActions(inputYaml string) (string, bool, error) {
	workflow := Workflow{}
	updated := false
	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return inputYaml, updated, fmt.Errorf("unable to parse yaml %v", err)
	}

	out := inputYaml

	for jobName, job := range workflow.Jobs {

		for _, step := range job.Steps {
			if len(step.Uses) > 0 {
				localUpdated := false
				out, localUpdated = pinAction(step.Uses, jobName, out)
				updated = updated || localUpdated
			}
		}
	}

	return out, updated, nil
}

func pinAction(action, jobName, inputYaml string) (string, bool) {

	updated := false
	if !strings.Contains(action, "@") || strings.HasPrefix(action, "docker://") {
		return inputYaml, updated // Cannot pin local actions and docker actions
	}

	leftOfAt := strings.Split(action, "@")
	tagOrBranch := leftOfAt[1]

	splitOnSlash := strings.Split(leftOfAt[0], "/")
	owner := splitOnSlash[0]
	repo := splitOnSlash[1]

	PAT := os.Getenv("PAT")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: PAT},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	ref, _, err := client.Git.GetRef(context.Background(), owner, repo, fmt.Sprintf("tags/%s", tagOrBranch))

	if err != nil {
		ref, _, err = client.Git.GetRef(context.Background(), owner, repo, fmt.Sprintf("heads/%s", tagOrBranch))

		if err != nil {
			// TODO: Log the error
			return inputYaml, updated
		}
	}

	commitSHA := ref.Object.SHA
	pinnedAction := fmt.Sprintf("%s@%s", leftOfAt[0], *commitSHA)
	updated = !strings.EqualFold(action, pinnedAction)
	inputYaml = strings.ReplaceAll(inputYaml, action, pinnedAction)
	return inputYaml, updated
}
