package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
)

const (
	kblabel           = "knowledge-base"
	stepsecurityowner = "step-security"
	stepsecurityrepo  = "secure-workflows"
	allIssues         = "all"
)

type GitHubJobStepOut struct {
	Action string
}

func CreateIssue(step *GitHubJobStepOut, owner, repo, runid string) (int, error) {
	// is action
	if len(step.Action) > 0 {
		// is kb not found
		_, err := GetActionKnowledgeBase(step.Action)

		if err != nil {
			// does issue already exist?
			issue, err := getExistingIssue(step.Action)

			if err != nil {
				return 0, err
			}

			if issue == nil {
				issue, err = createIssue(step, owner, repo, runid)

				if err != nil {
					fmt.Printf("[CreateIssue] error in creating issue for action %s: %v", step.Action, err)
					return 0, err
				}

				fmt.Printf("[CreateIssue] Issue created for action %s: %d", step.Action, issue.Number)
				return *issue.Number, nil
			} else {
				return *issue.Number, nil
			}
		} else {
			return 0, fmt.Errorf("action already has kb")
		}
	}

	return 0, fmt.Errorf("step is not an action")
}

func createIssue(step *GitHubJobStepOut, owner, repo, runid string) (*github.Issue, error) {
	PAT := os.Getenv("PAT")
	if PAT == "" {
		return nil, fmt.Errorf("[createIssue] PAT not set in env variable")
	}
	client := getClient(PAT)
	title := fmt.Sprintf("[KB] Add KB for %s", step.Action)
	labels := []string{kblabel}
	bodyLines := []string{}
	bodyLines = append(bodyLines, "harden-runner-link: ")
	bodyLines = append(bodyLines, fmt.Sprintf("https://app.stepsecurity.io/github/%s/%s/actions/runs/%s", owner, repo, runid))
	body := strings.Join(bodyLines, "\r\n")
	issue, _, err := client.Issues.Create(context.Background(), stepsecurityowner, stepsecurityrepo, &github.IssueRequest{Title: &title, Labels: &labels, Body: &body})

	if err != nil {
		return nil, err
	}

	return issue, nil
}

func getClient(PAT string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: PAT},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return client
}

func getExistingIssue(action string) (*github.Issue, error) {
	PAT := os.Getenv("PAT")
	if PAT == "" {
		return nil, fmt.Errorf("[createIssue] PAT not set in env variable")
	}

	client := getClient(PAT)

	issues, _, err := client.Issues.ListByRepo(context.Background(), stepsecurityowner, stepsecurityrepo,
		&github.IssueListByRepoOptions{Labels: []string{kblabel}, State: allIssues, ListOptions: github.ListOptions{PerPage: 100}})

	if err != nil {
		return nil, err
	}

	for _, issue := range issues {
		if strings.Contains(*issue.Title, action) {
			return issue, nil
		}
	}

	return nil, nil
}
