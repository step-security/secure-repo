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

	if isAbsolute(action) {
		return inputYaml, updated
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

	commitSHA, _, err := client.Repositories.GetCommitSHA1(ctx, owner, repo, tagOrBranch, "")
	if err != nil {
		return inputYaml, updated
	}

	pinnedAction := fmt.Sprintf("%s@%s", leftOfAt[0], commitSHA)
	updated = !strings.EqualFold(action, pinnedAction)
	inputYaml = strings.ReplaceAll(inputYaml, action, pinnedAction)
	return inputYaml, updated
}

// https://github.com/sethvargo/ratchet/blob/3524c5cfde0439099b3a37274e683af4c779b0d1/parser/refs.go#L56
func isAbsolute(ref string) bool {
	parts := strings.Split(ref, "@")
	last := parts[len(parts)-1]

	if len(last) == 40 && isAllHex(last) {
		return true
	}

	if len(last) == 71 && last[:6] == "sha256" && isAllHex(last[7:]) {
		return true
	}

	return false
}

// isAllHex returns true if the given string is all hex characters, false
// otherwise.
func isAllHex(s string) bool {
	for _, ch := range s {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
			return false
		}
	}
	return true
}
