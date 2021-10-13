package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"gopkg.in/yaml.v3"
)

type FixWorkflowPermsReponse struct {
	FinalOutput           string
	IsChanged             bool
	HasErrors             bool
	AlreadyHasPermissions bool
	IncorrectYaml         bool
	//	JobFixes              map[string]string
	JobErrors      map[string][]string
	MissingActions []string
}

const errorSecretInRunStep = "KnownIssue-1: Jobs with run steps that use token are not supported"
const errorSecretInRunStepEnvVariable = "KnownIssue-2: Jobs with run steps that use token in environment variable are not supported"
const errorLocalAction = "KnownIssue-3: Action %s is a local action. Local actions are not supported"
const errorMissingAction = "KnownIssue-4: Jobs with action %s are not supported"
const errorAlreadyHasPermissions = "KnownIssue-5: Workflows or Jobs that already have permissions are not modified"
const errorIncorrectYaml = "Unable to parse the YAML workflow file"

func alreadyHasJobPermissions(job Job) bool {
	if job.Permissions.Actions != "" ||
		job.Permissions.Checks != "" ||
		job.Permissions.Contents != "" ||
		job.Permissions.Deployments != "" ||
		job.Permissions.Issues != "" ||
		job.Permissions.Packages != "" ||
		job.Permissions.PullRequests != "" ||
		job.Permissions.Statuses != "" {
		return true
	}

	return false
}

func alreadyHasWorkflowPermissions(workflow Workflow) bool {
	if workflow.Permissions.Actions != "" ||
		workflow.Permissions.Checks != "" ||
		workflow.Permissions.Contents != "" ||
		workflow.Permissions.Deployments != "" ||
		workflow.Permissions.Issues != "" ||
		workflow.Permissions.Packages != "" ||
		workflow.Permissions.PullRequests != "" ||
		workflow.Permissions.Statuses != "" {
		return true
	}

	return false
}

func FixWorkflowPermissions(inputYaml string, svc dynamodbiface.DynamoDBAPI) (*FixWorkflowPermsReponse, error) {

	workflow := Workflow{}
	errors := make(map[string][]string)
	//fixes := make(map[string]string)
	fixWorkflowPermsReponse := &FixWorkflowPermsReponse{}

	actionPermissions, err := getActionPermissions(svc)

	if err != nil {
		return nil, fmt.Errorf("unable to read action permissions, %v", err)
	}

	err = yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		fixWorkflowPermsReponse.HasErrors = true
		fixWorkflowPermsReponse.IncorrectYaml = true
		return fixWorkflowPermsReponse, nil
	}

	if alreadyHasWorkflowPermissions(workflow) {
		// We are not modifying permissions if already defined
		fixWorkflowPermsReponse.HasErrors = true
		fixWorkflowPermsReponse.AlreadyHasPermissions = true
		return fixWorkflowPermsReponse, nil
	}

	out := inputYaml

	for jobName, job := range workflow.Jobs {

		if alreadyHasJobPermissions(job) {
			// We are not modifying permissions if already defined
			fixWorkflowPermsReponse.HasErrors = true
			fixWorkflowPermsReponse.AlreadyHasPermissions = true

			errors[jobName] = append(errors[jobName], errorAlreadyHasPermissions)
			continue
		}

		jobState := &JobState{ActionPermissions: actionPermissions}
		perms, err := jobState.getPermissions(job.Steps)

		if err != nil {
			for _, err := range jobState.Errors {
				errors[jobName] = append(errors[jobName], err.Error())
			}

			fixWorkflowPermsReponse.HasErrors = true
			fixWorkflowPermsReponse.MissingActions = append(fixWorkflowPermsReponse.MissingActions, jobState.MissingActions...)
			continue // skip fixing this job
		} else {
			if strings.Compare(inputYaml, fixWorkflowPermsReponse.FinalOutput) != 0 {
				fixWorkflowPermsReponse.IsChanged = true

				// This is to add on the fixes for jobs
				out, err = addPermissions(out, jobName, perms)

				if err != nil {
					// This should not happen
					return nil, err
				}
				// This is to get fix relative to original file
				//fixes[jobName], _ = addPermissions(inputYaml, jobName, perms)
			}
		}

	}
	fixWorkflowPermsReponse.FinalOutput = out
	//fixWorkflowPermsReponse.JobFixes = fixes
	fixWorkflowPermsReponse.JobErrors = errors

	return fixWorkflowPermsReponse, nil
}

func isGitHubToken(literal string) bool {
	literal = strings.ToLower(literal)
	if strings.Contains(literal, "secrets.github_token") || strings.Contains(literal, "github.token") {
		return true
	}

	return false
}

func (jobState *JobState) getPermissionsForAction(action Step) ([]string, error) {
	permissions := []string{}
	atIndex := strings.Index(action.Uses, "@")

	if atIndex == -1 {
		return nil, fmt.Errorf(errorLocalAction, action.Uses)
	}

	actionKey := action.Uses[0:atIndex]

	actionKey = strings.ReplaceAll(actionKey, "/", "-")

	actionKey = strings.ToLower(actionKey)

	actionPermission, isFound := jobState.ActionPermissions.Actions[actionKey]

	if isFound {

		// See elgohr-publish-docker-github-action.yml
		if actionKey == "elgohr-publish-docker-github-action" && action.With["registry"] != "" && strings.Contains(action.With["registry"], ".pkg.github.com") {
			if action.With["password"] != "" && strings.Contains(action.With["password"], "secrets.GITHUB_TOKEN") {
				permissions = append(permissions, "packages: write")
			}
		}

		if actionKey == "js-devtools-npm-publish" && action.With["registry"] != "" && strings.Contains(action.With["registry"], ".pkg.github.com") {
			if action.With["token"] != "" && strings.Contains(action.With["token"], "secrets.GITHUB_TOKEN") {
				permissions = append(permissions, "packages: write")
			}
		}

		if (actionKey == "brandedoutcast-publish-nuget" || actionKey == "rohith-publish-nuget") && action.With["NUGET_SOURCE"] != "" && strings.Contains(action.With["NUGET_SOURCE"], ".pkg.github.com") {
			if action.With["NUGET_KEY"] != "" && strings.Contains(action.With["NUGET_KEY"], "secrets.GITHUB_TOKEN") {
				permissions = append(permissions, "packages: write")
			}
		}

		if actionKey == "borales-actions-yarn" && action.With["registry-url"] != "" && strings.Contains(action.With["registry-url"], ".pkg.github.com") {
			if action.With["auth-token"] != "" && strings.Contains(action.With["auth-token"], "secrets.GITHUB_TOKEN") {
				permissions = append(permissions, "packages: write")
			}
		}

		// See docker-login-action.yml
		if actionKey == "docker-login-action" || actionKey == "azure-docker-login" {
			if action.With["password"] != "" && strings.Contains(action.With["password"], "secrets.GITHUB_TOKEN") {
				permissions = append(permissions, "packages: write")
			}
		}

		// Set registry info. This is for setup-node action. See action-setupnode-install-gpr.yml
		if actionKey == "actions-setup-node" && action.With["registry-url"] != "" && strings.Contains(action.With["registry-url"], ".pkg.github.com") {
			jobState.CurrentNpmPackageRegistry = action.With["registry-url"]
		}

		// Set registry info. This is for setup-dotnet action. See action-setupdotnet-publish-gpr.yml
		if actionKey == "actions-setup-dotnet" {
			if action.With["source-url"] != "" {
				jobState.CurrentNuGetSourceURL = action.With["source-url"]
			}
			if action.Env["NUGET_AUTH_TOKEN"] != "" {
				jobState.CurrentNugetAuthToken = action.Env["NUGET_AUTH_TOKEN"]
			}
		}

		// If action has a default token, and the token was set explicitly, but not to the Github token, no permissions are needed
		// See action-with-nondefault-token.yml as an example
		if actionPermission.DefaultToken != "" {
			if action.With[actionPermission.DefaultToken] != "" && !isGitHubToken(action.With[actionPermission.DefaultToken]) {
				return permissions, nil
			}
		}

		// If action expects token in env variable, and the token was not set, or not to the Github token, no permissions are needed
		// See action-envkey-non-ghtoken.yml as an example
		if actionPermission.EnvKey != "" {
			if action.Env[actionPermission.EnvKey] == "" || !isGitHubToken(action.Env[actionPermission.EnvKey]) {
				return permissions, nil
			}
		}

		// These are in ascending order, contents, then issues
		if actionPermission.Permissions.Actions != "" {
			permissions = append(permissions, "actions: "+actionPermission.Permissions.Actions)
		}

		if actionPermission.Permissions.Checks != "" {
			permissions = append(permissions, "checks: "+actionPermission.Permissions.Checks)
		}

		if actionPermission.Permissions.Contents != "" {
			permissions = append(permissions, "contents: "+actionPermission.Permissions.Contents)
		}

		if actionPermission.Permissions.Deployments != "" {
			permissions = append(permissions, "deployments: "+actionPermission.Permissions.Deployments)
		}

		if actionPermission.Permissions.Issues != "" {
			permissions = append(permissions, "issues: "+actionPermission.Permissions.Issues)
		}

		if actionPermission.Permissions.Packages != "" {
			permissions = append(permissions, "packages: "+actionPermission.Permissions.Packages)
		}

		if actionPermission.Permissions.PullRequests != "" {
			permissions = append(permissions, "pull-requests: "+actionPermission.Permissions.PullRequests)
		}

		if actionPermission.Permissions.SecurityEvents != "" {
			permissions = append(permissions, "security-events: "+actionPermission.Permissions.SecurityEvents)
		}

		if actionPermission.Permissions.Statuses != "" {
			permissions = append(permissions, "statuses: "+actionPermission.Permissions.Statuses)
		}

	} else {
		jobState.MissingActions = append(jobState.MissingActions, action.Uses)
		return nil, fmt.Errorf(errorMissingAction, action.Uses)
	}

	return permissions, nil
}

type JobState struct {
	CurrentNpmPackageRegistry string
	CurrentNuGetSourceURL     string
	CurrentNugetAuthToken     string

	MissingActions    []string
	Errors            []error
	ActionPermissions *ActionPermissions
}

func evaluateEnvironmentVariables(step Step) string {
	keyToEvaluate := ""
	run := step.Run
	for key, value := range step.Env {
		if strings.Contains(value, "secrets.GITHUB_TOKEN") || strings.Contains(value, "github.token") {
			keyToEvaluate = key
			break
		}
	}

	if keyToEvaluate != "" {
		run = strings.ReplaceAll(run, fmt.Sprintf("${%s}", keyToEvaluate), "${{ secrets.GITHUB_TOKEN }}")
	}

	return run
}

func (jobState *JobState) getPermissionsForRunStep(step Step) ([]string, error) {
	permissions := []string{}

	runStep := evaluateEnvironmentVariables(step)

	// reviewdog
	if step.Env["REVIEWDOG_GITHUB_API_TOKEN"] != "" && isGitHubToken(step.Env["REVIEWDOG_GITHUB_API_TOKEN"]) {
		if strings.Contains(runStep, "reviewdog") {
			permissions = append(permissions, "checks: write")
			permissions = append(permissions, "pull-requests: write")
			return permissions, nil
		}
	}

	// if it is a run step and has set node token, we may need to give packages permission
	if step.Env["NODE_AUTH_TOKEN"] != "" && (strings.Contains(step.Env["NODE_AUTH_TOKEN"], "secrets.GITHUB_TOKEN") || strings.Contains(step.Env["NODE_AUTH_TOKEN"], "github.token")) && strings.Contains(jobState.CurrentNpmPackageRegistry, "npm.pkg.github.com") {
		if strings.Contains(runStep, "install") {
			permissions = append(permissions, "packages: read")
			return permissions, nil
		} else if strings.Contains(runStep, "publish") {
			permissions = append(permissions, "packages: write")
			return permissions, nil
		}
	}

	// Dotnet. See action-setupdotnet-publish-gpr test case
	if strings.Contains(runStep, "dotnet nuget push") {
		// No token is explicitly provided
		if !strings.Contains(runStep, "-k") && !strings.Contains(runStep, "--api-key") {
			// If setup-dotnet action has set the source url and token already
			if strings.Contains(jobState.CurrentNuGetSourceURL, "pkg.github.com") && (strings.Contains(jobState.CurrentNugetAuthToken, "secrets.GITHUB_TOKEN") || strings.Contains(jobState.CurrentNugetAuthToken, "github.token")) {

				permissions = append(permissions, "packages: write")
				return permissions, nil
			}
			// If current step has env
			if step.Env["NUGET_AUTH_TOKEN"] != "" && (strings.Contains(step.Env["NUGET_AUTH_TOKEN"], "secrets.GITHUB_TOKEN") || strings.Contains(step.Env["NUGET_AUTH_TOKEN"], "github.token")) {

				permissions = append(permissions, "packages: write")
				return permissions, nil
			}
		} else if strings.Contains(runStep, "-k ${{ secrets.GITHUB_TOKEN }}") || strings.Contains(runStep, "-k ${{ github.token }}") || strings.Contains(runStep, "--api-key ${{ secrets.GITHUB_TOKEN }}") || strings.Contains(runStep, "--api-key ${{ github.token }}") {
			permissions = append(permissions, "packages: write")
			return permissions, nil
		}
	}

	// Dotnet. See action-setupdotnet-publish-curl test case
	if strings.Contains(runStep, "curl") && strings.Contains(runStep, "PUT") {
		if (strings.Contains(runStep, "secrets.GITHUB_TOKEN") || strings.Contains(runStep, "github.token")) && strings.Contains(runStep, "nuget.pkg.github.com") {
			permissions = append(permissions, "packages: write")
			return permissions, nil
		}
	}

	// Git push/ apply. See content-write-run-step.yml
	if strings.Contains(runStep, "git apply") || strings.Contains(runStep, "git push") {
		permissions = append(permissions, "contents: write")
		return permissions, nil
	}

	// Java. See action-setup-java.yml
	if strings.Contains(runStep, "gradle publish") || strings.Contains(runStep, "mvn deploy") {

		// if any of the environment variables have the github token
		for _, value := range step.Env {
			if strings.Contains(value, "secrets.GITHUB_TOKEN") || strings.Contains(value, "github.token") {
				permissions = append(permissions, "packages: write")
				return permissions, nil
			}
		}
	}

	if strings.Contains(runStep, "secrets.GITHUB_TOKEN") || strings.Contains(runStep, "github.token") {
		return nil, fmt.Errorf(errorSecretInRunStep)
	}

	for _, envValue := range step.Env {

		if strings.Contains(envValue, "secrets.GITHUB_TOKEN") || strings.Contains(envValue, "github.token") {
			return nil, fmt.Errorf(errorSecretInRunStepEnvVariable)
		}

	}

	return permissions, nil
}

func (jobState *JobState) getPermissions(steps []Step) ([]string, error) {
	permissions := []string{}

	for _, step := range steps {

		if step.Uses != "" { // it is an action
			permsForAction, err := jobState.getPermissionsForAction(step)

			if err != nil {
				jobState.Errors = append(jobState.Errors, err)
			}

			permissions = append(permissions, permsForAction...)
		} else if step.Run != "" { // if it is a run step
			permsForRunStep, err := jobState.getPermissionsForRunStep(step)

			if err != nil {
				jobState.Errors = append(jobState.Errors, err)
			}

			permissions = append(permissions, permsForRunStep...)
		}

	}

	if len(jobState.Errors) > 0 {
		return nil, fmt.Errorf("Job has errors")
	}

	if len(permissions) == 0 {
		return []string{"contents: none"}, nil
	}

	permissions = removeRedundantPermisions(permissions)

	return permissions, nil
}

func removeRedundantPermisions(permissions []string) []string {

	permissions = removeDuplicates(permissions)
	var newPermissions []string
	// if there is read and write of same permissions, e.g contents: read and contents: write, then contents: read should be removed
	permMap := make(map[string]string)

	for _, perm := range permissions {
		permSplit := strings.Split(perm, ":")

		permInMap, found := permMap[permSplit[0]]

		if found {
			if permInMap == " read" {
				permMap[permSplit[0]] = permSplit[1] // this should be a write
			}

		} else {
			permMap[permSplit[0]] = permSplit[1]
		}
	}

	for k, v := range permMap {
		newPermissions = append(newPermissions, k+":"+v)
	}

	sort.Strings(newPermissions)
	return newPermissions
}

func addPermissions(inputYaml string, jobName string, permissions []string) (string, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", fmt.Errorf("unable to parse yaml %v", err)
	}

	jobNode := iterateNode(&t, jobName)

	if jobNode == nil {
		return "", fmt.Errorf("jobName %s not found in the input yaml", jobName)
	}

	inputLines := strings.Split(inputYaml, "\n")
	var output []string
	for i := 0; i < jobNode.Line-1; i++ {
		output = append(output, inputLines[i])
	}

	spaces := ""
	for i := 0; i < jobNode.Column-1; i++ {
		spaces += " "
	}

	output = append(output, spaces+"permissions:")

	for _, perm := range permissions {
		output = append(output, spaces+"  "+perm)
	}

	for i := jobNode.Line - 1; i < len(inputLines); i++ {
		output = append(output, inputLines[i])
	}

	return strings.Join(output, "\n"), nil
}

// iterateNode will recursive look for the node following the identifier Node,
// as go-yaml has a node for the key and the value itself
// we want to manipulate the value Node
func iterateNode(node *yaml.Node, identifier string) *yaml.Node {
	returnNode := false
	for _, n := range node.Content {
		if n.Value == identifier {
			returnNode = true
			continue
		}
		if returnNode {
			if n.Tag == "!!map" {
				return n
			} else {
				returnNode = false
			}
		}
		if len(n.Content) > 0 {
			ac_node := iterateNode(n, identifier)
			if ac_node != nil {
				return ac_node
			}
		}
	}
	return nil
}
