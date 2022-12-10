package workflow

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
)

const CodeQLWorkflowFileName = "codeql.yml"

type WorkflowParameters struct {
	LanguagesToAdd []string
	DefaultBranch  string
}

func getTemplate(file string) (string, error) {
	templatePath := os.Getenv("WORKFLOW-TEMPLATE")

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
	if name == "codeql" {
		codeqlWorkflow, err := getTemplate(CodeQLWorkflowFileName)
		if err != nil {
			return "", err
		}

		sort.Strings(workflowParameters.LanguagesToAdd)
		codeqlWorkflow = strings.ReplaceAll(codeqlWorkflow, "$default-branch", fmt.Sprintf(`"%s"`, workflowParameters.DefaultBranch))
		codeqlWorkflow = strings.ReplaceAll(codeqlWorkflow, "$detected-codeql-languages", strings.Join(workflowParameters.LanguagesToAdd, ", "))
		codeqlWorkflow = strings.ReplaceAll(codeqlWorkflow, "$cron-weekly", fmt.Sprintf(`"%s"`, "0 0 * * 1")) // Note: Runs every monday at 12:00 AM

		return codeqlWorkflow, nil
	} else {
		return "", fmt.Errorf("match for %s Workflow name not found", name)
	}
}
