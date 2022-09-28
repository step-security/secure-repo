package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PaesslerAG/gval"
	"github.com/generikvault/gvalstrings"
	"gopkg.in/yaml.v3"
)

type SecureWorkflowReponse struct {
	OriginalInput         string
	FinalOutput           string
	IsChanged             bool
	HasErrors             bool
	AlreadyHasPermissions bool
	PinnedActions         bool
	AddedHardenRunner     bool
	AddedPermissions      bool
	IncorrectYaml         bool
	WorkflowFetchError    bool
	JobErrors             []JobError
	MissingActions        []string
}

type JobError struct {
	JobName string
	Errors  []string
}

const errorSecretInRunStep = "KnownIssue-1: Jobs with run steps that use token are not supported"
const errorSecretInRunStepEnvVariable = "KnownIssue-2: Jobs with run steps that use token in environment variable are not supported"
const errorLocalAction = "KnownIssue-3: Action %s is a local action. Local actions are not supported"
const errorMissingAction = "KnownIssue-4: Action %s is not in the knowledge base"
const errorAlreadyHasPermissions = "KnownIssue-5: Permissions were not added to the job since it already had permissions defined"
const errorDockerAction = "KnownIssue-6: Action %s is a docker action which uses Github token. Docker actions that uses token are not supported"
const errorReusableWorkflow = "KnownIssue-7: Action %s is a reusable workflow. Reusable workflows are not supported as of now."
const errorIncorrectYaml = "Unable to parse the YAML workflow file"

//To avoid a typo while adding the permissions
const (
	actions_read              = "actions: read"
	actions_write             = "actions: write"
	checks_read               = "checks: read"
	checks_write              = "checks: write"
	contents_read             = "contents: read"
	contents_write            = "contents: write"
	deployments_read          = "deployments: read"
	deployments_write         = "deployments: write"
	id_token_read             = "id-token: read"
	id_token_write            = "id-token: write"
	issues_read               = "issues: read"
	issues_write              = "issues: write"
	discussions_read          = "discussions: read"
	discussions_write         = "discussions: write"
	packages_read             = "packages: read"
	packages_write            = "packages: write"
	pages_read                = "pages: read"
	pages_write               = "pages: write"
	pull_requests_read        = "pull-requests: read"
	pull_requests_write       = "pull-requests: write"
	repository_projects_read  = "repository-projects: read"
	repository_projects_write = "repository-projects: write"
	security_events_read      = "security-events: read"
	security_events_write     = "security-events: write"
	statuses_read             = "statuses: read"
	statuses_write            = "statuses: write"
)

func alreadyHasJobPermissions(job Job) bool {
	return job.Permissions.IsSet
}

func alreadyHasWorkflowPermissions(workflow Workflow) bool {
	return workflow.Permissions.IsSet
}

func AddWorkflowLevelPermissions(inputYaml string, addProjectComment bool) (string, error) {
	workflow := Workflow{}

	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return "", err
	}

	if alreadyHasWorkflowPermissions(workflow) {
		// We are not modifying permissions if already defined
		return inputYaml, fmt.Errorf("Workflow already has permissions")
	}

	t := yaml.Node{}

	err = yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return inputYaml, fmt.Errorf("unable to parse yaml %v", err)
	}

	line := 0
	column := 0
	topNode := t.Content
	for _, n := range topNode[0].Content {
		if n.Value == "jobs" && n.Tag == "!!str" {
			line = n.Line
			column = n.Column
			break
		}
	}

	if line == 0 {
		return inputYaml, fmt.Errorf("jobs not found in workflow")
	}

	inputLines := strings.Split(inputYaml, "\n")
	var output []string
	for i := 0; i < line-1; i++ {
		output = append(output, inputLines[i])
	}

	spaces := ""
	for i := 0; i < column-1; i++ {
		spaces += " "
	}

	if addProjectComment {
		output = append(output, spaces+"permissions:  # added using https://github.com/step-security/secure-workflows")
	} else {
		output = append(output, spaces+"permissions:")
	}

	output = append(output, spaces+"  contents: read")
	output = append(output, "")

	for i := line - 1; i < len(inputLines); i++ {
		output = append(output, inputLines[i])
	}

	return strings.Join(output, "\n"), nil
}

func AddJobLevelPermissions(inputYaml string) (*SecureWorkflowReponse, error) {

	workflow := Workflow{}
	errors := make(map[string][]string)
	//fixes := make(map[string]string)
	fixWorkflowPermsReponse := &SecureWorkflowReponse{}

	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		fixWorkflowPermsReponse.HasErrors = true
		fixWorkflowPermsReponse.IncorrectYaml = true
		fixWorkflowPermsReponse.FinalOutput = inputYaml
		return fixWorkflowPermsReponse, nil
	}

	if alreadyHasWorkflowPermissions(workflow) {
		// We are not modifying permissions if already defined
		fixWorkflowPermsReponse.HasErrors = true
		fixWorkflowPermsReponse.AlreadyHasPermissions = true
		fixWorkflowPermsReponse.FinalOutput = inputYaml
		return fixWorkflowPermsReponse, nil
	}

	out := inputYaml

	for jobName, job := range workflow.Jobs {

		if alreadyHasJobPermissions(job) {
			// We are not modifying permissions if already defined
			fixWorkflowPermsReponse.HasErrors = true
			errors[jobName] = append(errors[jobName], errorAlreadyHasPermissions)
			continue
		}

		if IsCallingReusableWorkflow(job) {
			fixWorkflowPermsReponse.HasErrors = true
			errors[jobName] = append(errors[jobName], fmt.Sprintf(errorReusableWorkflow, job.Uses))
			continue
		}

		jobState := &JobState{}
		jobState.WorkflowEnv = workflow.Env
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

				if len(perms) == 1 && strings.Contains(perms[0], contents_read) {
					// Don't add contents: read, because it will get defined at workflow level
					continue
				} else {
					// This is to add on the fixes for jobs
					out, err = addPermissions(out, jobName, perms)

					if err != nil {
						// This should not happen
						return nil, err
					}
				}

			}
		}

	}
	fixWorkflowPermsReponse.FinalOutput = out

	// Convert to array of JobError from map
	for job, jobErrors := range errors {
		jobError := JobError{JobName: job}
		jobError.Errors = append(jobError.Errors, jobErrors...)

		fixWorkflowPermsReponse.JobErrors = append(fixWorkflowPermsReponse.JobErrors, jobError)
	}

	return fixWorkflowPermsReponse, nil
}

func isGitHubToken(literal string) bool {
	literal = strings.ToLower(literal)
	literal = strings.ReplaceAll(literal, "${{", "")
	literal = strings.ReplaceAll(literal, "}}", "")
	literal = strings.Trim(literal, " ")

	if literal == "secrets.github_token" || literal == "github.token" {
		return true
	}

	return false
}

func (jobState *JobState) getPermissionsForAction(action Step) ([]string, error) {
	permissions := []string{}
	atIndex := strings.Index(action.Uses, "@")

	//Do not check for permissions in KB, if it is a docker action
	if strings.HasPrefix(action.Uses, "docker://") {
		//Return error if it uses token in environment variable
		for _, envValue := range action.Env {
			if isGitHubToken(envValue) {
				return nil, fmt.Errorf(errorDockerAction, action.Uses)
			}
		}
		//Return error if it uses token in action input
		for _, actionValue := range action.With {
			if isGitHubToken(actionValue) {
				return nil, fmt.Errorf(errorDockerAction, action.Uses)
			}
		}
		//else return without raising error
		return permissions, nil
	}

	if atIndex == -1 {
		return nil, fmt.Errorf(errorLocalAction, action.Uses)
	}

	actionKey := action.Uses[0:atIndex]

	actionMetadata, err := GetActionKnowledgeBase(actionKey)

	if err != nil {
		jobState.MissingActions = append(jobState.MissingActions, action.Uses)
		return nil, fmt.Errorf(errorMissingAction, action.Uses)
	}

	// If action has a default token, and the token was set explicitly, but not to the Github token, no permissions are needed
	if actionMetadata.GitHubToken.ActionInput.IsDefault {
		if action.With[actionMetadata.GitHubToken.ActionInput.Input] != "" && !isGitHubToken(action.With[actionMetadata.GitHubToken.ActionInput.Input]) {
			return permissions, nil
		}
	}

	// If action has does not have a default token, and the token was not set explicitly, no permissions are needed
	if actionMetadata.GitHubToken.ActionInput.Input != "" && !actionMetadata.GitHubToken.ActionInput.IsDefault {
		if action.With[actionMetadata.GitHubToken.ActionInput.Input] == "" || !isGitHubToken(action.With[actionMetadata.GitHubToken.ActionInput.Input]) {
			return permissions, nil
		}
	}

	// If action expects token in env variable, and the token was not set, or not to the Github token, no permissions are needed
	if actionMetadata.GitHubToken.EnvironmentVariableName != "" {
		if action.Env[actionMetadata.GitHubToken.EnvironmentVariableName] == "" || !isGitHubToken(action.Env[actionMetadata.GitHubToken.EnvironmentVariableName]) {
			return permissions, nil
		}
	}

	// TODO: Fix the order
	for scope, value := range actionMetadata.GitHubToken.Permissions.Scopes {
		if len(value.Expression) == 0 || evaluateExpression(value.Expression, action) {
			permissions = append(permissions, fmt.Sprintf("%s: %s  # for %s %s", scope, value.Permission, actionKey, value.Reason))
		}
	}

	return permissions, nil
}

func evaluateExpression(expression string, action Step) bool {
	vars := make(map[string]interface{})
	vars["with"] = action.With

	expression = strings.ReplaceAll(expression, "${{", "")
	expression = strings.ReplaceAll(expression, "}}", "")
	expression = strings.Trim(expression, " ")

	value, err := gval.Evaluate(expression,
		vars,
		gvalstrings.SingleQuoted(),
		gval.Function("contains", func(args ...interface{}) (interface{}, error) {
			switch v := args[0].(type) {
			case With:
				inputMap := v
				key := args[1].(string)
				_, found := inputMap[key]
				return (bool)(found), nil
			}
			return nil, fmt.Errorf("type not supported %T", args[0])
		}))
	if err != nil {
		return false
	}

	return value.(bool)
}

type JobState struct {
	CurrentNpmPackageRegistry string
	CurrentNuGetSourceURL     string
	CurrentNugetAuthToken     string

	WorkflowEnv       map[string]string // map of workflow level environment variables
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

type Permission struct {
	permission string
	action     string
	reason     string
}

func (jobState *JobState) getPermissionsForRunStep(step Step) ([]Permission, error) {
	permissions := []Permission{}

	runStep := evaluateEnvironmentVariables(step)

	//Note: Add permissions in this format
	//permissions = append(permissions, Permission{permission: "", action: "", reason: ""})

	// reviewdog
	if step.Env["REVIEWDOG_GITHUB_API_TOKEN"] != "" && isGitHubToken(step.Env["REVIEWDOG_GITHUB_API_TOKEN"]) {
		if strings.Contains(runStep, "reviewdog") {
			permissions = append(permissions, Permission{permission: checks_write, action: "reviewdog", reason: "to create check"})
			permissions = append(permissions, Permission{permission: pull_requests_write, action: "reviewdog", reason: "to add comment to the PR"})
			return permissions, nil
		}
	}

	// if it is a run step and has set node token, we may need to give packages permission
	if step.Env["NODE_AUTH_TOKEN"] != "" && (strings.Contains(step.Env["NODE_AUTH_TOKEN"], "secrets.GITHUB_TOKEN") || strings.Contains(step.Env["NODE_AUTH_TOKEN"], "github.token")) && strings.Contains(jobState.CurrentNpmPackageRegistry, "npm.pkg.github.com") {
		if strings.Contains(runStep, "install") {
			permissions = append(permissions, Permission{permission: packages_read, action: "node", reason: "to install packages"})
			return permissions, nil
		} else if strings.Contains(runStep, "publish") {
			permissions = append(permissions, Permission{permission: packages_write, action: "node", reason: "to publish packages"})
			return permissions, nil
		}
	}

	// Dotnet. See action-setupdotnet-publish-gpr test case
	if strings.Contains(runStep, "dotnet nuget push") {
		// No token is explicitly provided
		if !strings.Contains(runStep, "-k") && !strings.Contains(runStep, "--api-key") {
			// If setup-dotnet action has set the source url and token already
			if strings.Contains(jobState.CurrentNuGetSourceURL, "pkg.github.com") && (strings.Contains(jobState.CurrentNugetAuthToken, "secrets.GITHUB_TOKEN") || strings.Contains(jobState.CurrentNugetAuthToken, "github.token")) {

				permissions = append(permissions, Permission{permission: packages_write, action: "setup-dotnet", reason: "to create package"})
				return permissions, nil
			}
			// If current step has env
			if step.Env["NUGET_AUTH_TOKEN"] != "" && (strings.Contains(step.Env["NUGET_AUTH_TOKEN"], "secrets.GITHUB_TOKEN") || strings.Contains(step.Env["NUGET_AUTH_TOKEN"], "github.token")) {

				permissions = append(permissions, Permission{permission: packages_write, action: "setup-dotnet", reason: "to create package"})
				return permissions, nil
			}
		} else if strings.Contains(runStep, "-k ${{ secrets.GITHUB_TOKEN }}") || strings.Contains(runStep, "-k ${{ github.token }}") || strings.Contains(runStep, "--api-key ${{ secrets.GITHUB_TOKEN }}") || strings.Contains(runStep, "--api-key ${{ github.token }}") {
			permissions = append(permissions, Permission{permission: packages_write, action: "setup-dotnet", reason: "to create package"})
			return permissions, nil
		}
	}

	// Dotnet. See action-setupdotnet-publish-curl test case
	if strings.Contains(runStep, "curl") && strings.Contains(runStep, "PUT") {
		if (strings.Contains(runStep, "secrets.GITHUB_TOKEN") || strings.Contains(runStep, "github.token")) && strings.Contains(runStep, "nuget.pkg.github.com") {
			permissions = append(permissions, Permission{permission: packages_write, action: "setup-dotnet", reason: "to create package"})
			return permissions, nil
		}
	}

	// Git push. See content-write-run-step.yml
	if strings.Contains(runStep, "git push") {
		permissions = append(permissions, Permission{permission: contents_write, action: "Git", reason: "to git push"})
		return permissions, nil
	}

	if strings.Contains(runStep, "mkdocs gh-deploy") {
		permissions = append(permissions, Permission{permission: contents_write, action: "mkdocs gh-deploy", reason: "to publish docs"})
		return permissions, nil
	}

	// Java. See action-setup-java.yml
	if strings.Contains(runStep, "gradle publish") || strings.Contains(runStep, "mvn deploy") {

		// if any of the environment variables have the github token
		for _, value := range step.Env {
			if strings.Contains(value, "secrets.GITHUB_TOKEN") || strings.Contains(value, "github.token") {
				permissions = append(permissions, Permission{permission: packages_write, action: "setup-java", reason: "to create package"})
				return permissions, nil
			}
		}
	}

	// Dependabot run steps reference: https://github.com/dependabot/fetch-metadata#usage-instructions
	// Dependabot auto approve
	if step.Env["GITHUB_TOKEN"] != "" && isGitHubToken(step.Env["GITHUB_TOKEN"]) {
		if strings.Contains(runStep, "gh pr review --approve") {
			permissions = append(permissions, Permission{permission: pull_requests_write, action: "dependabot", reason: "to enable auto-approve"})
			return permissions, nil
		}
	}

	// Dependabot auto merge
	if step.Env["GITHUB_TOKEN"] != "" && isGitHubToken(step.Env["GITHUB_TOKEN"]) {
		if strings.Contains(runStep, "gh pr merge --auto --merge") {
			permissions = append(permissions, Permission{permission: contents_write, action: "dependabot", reason: "to enable auto-merge"})
			return permissions, nil
		}
	}

	// Dependabot auto label
	if step.Env["GITHUB_TOKEN"] != "" && isGitHubToken(step.Env["GITHUB_TOKEN"]) {
		if strings.Contains(runStep, "gh pr edit") && strings.Contains(runStep, "--add-label") {
			permissions = append(permissions, Permission{permission: repository_projects_write, action: "dependabot", reason: "to enable auto-label"})
			permissions = append(permissions, Permission{permission: issues_write, action: "dependabot", reason: "to enable auto-label"})
			return permissions, nil
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

			// Add workflow level env variables to each step
			for k, v := range jobState.WorkflowEnv {
				_, found := step.Env[k]
				if !found {
					if step.Env == nil {
						step.Env = make(map[string]string)
					}
					step.Env[k] = v
				}
			}

			permsForAction, err := jobState.getPermissionsForAction(step)

			if err != nil {
				jobState.Errors = append(jobState.Errors, err)
			}

			permissions = append(permissions, permsForAction...)
		} else if step.Run != "" { // if it is a run step
			RunStepPerms, err := jobState.getPermissionsForRunStep(step)
			if err != nil {
				jobState.Errors = append(jobState.Errors, err)
			}

			permsForRunStep := []string{}
			for runStep := range RunStepPerms {
				permsForRunStep = append(permsForRunStep, fmt.Sprintf("%s  # for %s %s", RunStepPerms[runStep].permission, RunStepPerms[runStep].action, RunStepPerms[runStep].reason))
			}

			permissions = append(permissions, permsForRunStep...)
		}

	}

	if len(jobState.Errors) > 0 {
		return nil, fmt.Errorf("Job has errors")
	}

	if len(permissions) == 0 {
		// changing to contents: read as this is not added to job level
		// if job needs no perm, it will not get anything adding at job level
		// workflow level will add contents: read
		// covered in test case job-level-none-perm.yml
		return []string{"contents: read"}, nil
	}

	permissions = removeRedundantPermisions(permissions)

	return permissions, nil
}

func removeRedundantPermisions(permissions []string) []string {

	permissions = removeDuplicates(permissions)
	/*
	 permissions would be like
	 contents: read # for actions/checkout to fetch code
	 pull-requests: write # for action/something to create PR
	*/
	var newPermissions []string
	// if there is read and write of same permissions, e.g contents: read and contents: write, then contents: read should be removed
	// key will be the scope, e.g. contents or pull-requests
	// value will be the value in permissions
	permMap := make(map[string]string)

	for _, perm := range permissions {
		permSplit := strings.Split(perm, ":")
		scope := permSplit[0]           // e.g. contents
		permWithComment := permSplit[1] // e.g. read # for actions/checkout to fetch code

		permInMap, found := permMap[scope]

		if found {
			scopeValue := strings.Trim(strings.Split(permInMap, "#")[0], " ") // e.g. read
			if scopeValue == "read" {
				permMap[scope] = permWithComment
			}

		} else {
			permMap[scope] = permWithComment
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

	jobNode := iterateNode(&t, jobName, "!!map", 0)

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

func iterateNode(node *yaml.Node, identifier, tag string, minLine int) *yaml.Node {
	returnNode := false
	for _, n := range node.Content {
		if n.Value == identifier {
			returnNode = true
			continue
		}
		if returnNode {
			if n.Tag == tag && n.Line > minLine {
				return n
			} else {
				returnNode = false
			}
		}
		if len(n.Content) > 0 {
			ac_node := iterateNode(n, identifier, tag, minLine)
			if ac_node != nil {
				return ac_node
			}
		}
	}
	return nil
}
