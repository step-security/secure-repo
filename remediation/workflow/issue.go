package workflow

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v40/github"
	metadata "github.com/step-security/secure-repo/remediation/workflow/metadata"
	"golang.org/x/oauth2"
)

const (
	kblabel           = "knowledge-base"
	stepsecurityowner = "step-security"
	branch            = "main"
	workflowFile      = "kbanalysis.yml"
	stepsecurityrepo  = "secure-repo"
	allIssues         = "all"
	openIssues        = "open"
)

func CreatePR(Action string) error {
	// is action
	if len(Action) > 0 {
		// is kb not found
		_, err := metadata.GetActionKnowledgeBase(Action)

		if err != nil {
			err = dispatchWorkflow(Action)
			return err
		} else {
			return fmt.Errorf("action already has kb")
		}
	}

	return fmt.Errorf("step is not an action")
}

type Repo struct {
	owner string
	repo  string
}

func getRepo(action string) Repo {
	r := Repo{}

	i := 0
	for i < len(action) {
		if action[i] == '/' {
			r.owner = action[0:i]
			break
		}
		i++
	}
	r.repo = action[i+1:]
	return r
}

func dispatchWorkflow(action string) error {
	PAT := os.Getenv("PAT")
	if PAT == "" {
		return fmt.Errorf("[dispatchWorkflow] PAT not set in env variable")
	}
	client := getClient(PAT)
	repos := getRepo(action)
	inputs := make(map[string]interface{})
	inputs["owner"] = repos.owner
	inputs["repo"] = repos.repo
	eventRequest := github.CreateWorkflowDispatchEventRequest{Ref: branch, Inputs: inputs}

	_, err := client.Actions.CreateWorkflowDispatchEventByFileName(context.Background(), stepsecurityowner, stepsecurityrepo, workflowFile, eventRequest)

	return err

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
