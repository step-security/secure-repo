package workflow

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
)

const (
	CodeQLWorkflowFileName   = "codeql.yml"
	DependencyReviewFileName = "dependency-review.yml"
	ScorecardFileName        = "scorecards.yml"
	CodeQL                   = "CodeQL"
	DependencyReview         = "Dependency-review"
	Scorecard                = "Scorecard"
)

type WorkflowParameters struct {
	LanguagesToAdd []string
	DefaultBranch  string
}

func getTemplate(file string) (string, error) {
	templatePath := os.Getenv("WORKFLOW_TEMPLATES")

	if templatePath == "" {
		templatePath = "../../workflow-templates"
	}

	template, err := ioutil.ReadFile(path.Join(templatePath, file))
	if err != nil {
		return "", err
	}

	return string(template), nil
}

func AddWorkflow(name string, workflowParameters WorkflowParameters) (string, error) {
	if name == CodeQL {
		codeqlWorkflow, err := getTemplate(CodeQLWorkflowFileName)
		if err != nil {
			return "", err
		}

		sort.Strings(workflowParameters.LanguagesToAdd)
		codeqlWorkflow = strings.ReplaceAll(codeqlWorkflow, "$default-branch", fmt.Sprintf(`"%s"`, workflowParameters.DefaultBranch))
		codeqlWorkflow = strings.ReplaceAll(codeqlWorkflow, "$detected-codeql-languages", strings.Join(workflowParameters.LanguagesToAdd, ", "))
		codeqlWorkflow = strings.ReplaceAll(codeqlWorkflow, "$cron-weekly", fmt.Sprintf(`"%s"`, "0 0 * * 1")) // Note: Runs every monday at 12:00 AM

		return codeqlWorkflow, nil

	} else if name == DependencyReview {
		dependencyReviewWorkflow, err := getTemplate(DependencyReviewFileName)
		if err != nil {
			return "", err
		}
		return dependencyReviewWorkflow, nil

	} else if name == Scorecard {
		scorecardsWorkflow, err := getTemplate(ScorecardFileName)
		if err != nil {
			return "", err
		}
		scorecardsWorkflow = strings.ReplaceAll(scorecardsWorkflow, "$default-branch", fmt.Sprintf(`"%s"`, workflowParameters.DefaultBranch))
		return scorecardsWorkflow, nil

	} else {
		return "", fmt.Errorf("match for %s Workflow name not found", name)
	}
}