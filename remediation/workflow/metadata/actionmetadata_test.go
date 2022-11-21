package metadata

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

func TestKnowledgeBase(t *testing.T) {
	kbFolder := os.Getenv("KBFolder")

	if kbFolder == "" {
		kbFolder = "../../../knowledge-base/actions"
	}

	lintIssues := []string{}

	err := filepath.Walk(kbFolder,
		func(filePath string, info os.FileInfo, err error) error {
			if !strings.HasSuffix(info.Name(), "yml") && !strings.HasSuffix(info.Name(), "yaml") {
				return nil
			}

			if err != nil {
				lintIssues = append(lintIssues, fmt.Sprintf("Error reading %s: %v", filePath, err))
				return nil
			}

			if strings.ToLower(filePath) != filePath {
				lintIssues = append(lintIssues, fmt.Sprintf("File path should be lowercase, not %s", filePath))
				return nil
			}

			if info.Name() != "action-security.yml" {
				lintIssues = append(lintIssues, fmt.Sprintf("File must be named action-security.yml, not %s at %s", info.Name(), filePath))
				return nil
			}

			// validating the action repo
			if !doesActionRepoExist(filePath) {
				lintIssues = append(lintIssues, fmt.Sprintf("Action repo does not exist at %s", filePath))
				return nil
			}

			input, err := ioutil.ReadFile(filePath)

			if err != nil {
				lintIssues = append(lintIssues, fmt.Sprintf("Unable to read action-security.yml at %s", filePath))
				return nil
			}

			actionMetadata := ActionMetadata{}

			err = yaml.Unmarshal([]byte(input), &actionMetadata)
			if err != nil {
				lintIssues = append(lintIssues, fmt.Sprintf("Unable to unmarshall action-security.yml at %s", filePath))
				return nil
			}

			if actionMetadata.Name == "" {
				lintIssues = append(lintIssues, fmt.Sprintf("Name must not be empty in action-security.yml at %s", filePath))
				return nil
			}

			for _, endpoint := range actionMetadata.AllowedEndpoints {
				if endpoint.FQDN == "" {
					lintIssues = append(lintIssues, fmt.Sprintf("FQDN must not be empty in action-security.yml at %s", filePath))
					return nil
				}

				if strings.ToLower(endpoint.FQDN) != endpoint.FQDN {
					lintIssues = append(lintIssues, fmt.Sprintf("FQDN must be all lower case in action-security.yml at %s", filePath))
					return nil
				}

				if endpoint.Port == 0 {
					lintIssues = append(lintIssues, fmt.Sprintf("Port must not be empty in action-security.yml at %s", filePath))
					return nil
				}

				if endpoint.Reason == "" {
					lintIssues = append(lintIssues, fmt.Sprintf("Reason must not be empty for fqdn %s in action-security.yml at %s", endpoint.FQDN, filePath))
					return nil
				}

				if !strings.HasPrefix(endpoint.Reason, "to ") {
					lintIssues = append(lintIssues, fmt.Sprintf("Reason must start with 'to '. It is currently %s in action-security.yml at %s", endpoint.Reason, filePath))
					return nil
				}

				if strings.HasSuffix(endpoint.Reason, ".") {
					lintIssues = append(lintIssues, fmt.Sprintf("Reason must not end with '.'. It is currently %s in action-security.yml at %s", endpoint.Reason, filePath))
					return nil
				}
			}

			//If permission does not exist, Env variable and Action input must not exist
			if len(actionMetadata.GitHubToken.Permissions.Scopes) == 0 {
				if actionMetadata.GitHubToken.EnvironmentVariableName != "" {
					lintIssues = append(lintIssues, fmt.Sprintf("Environment variable name must not exist when there is no permissions. It is currently set to '%s' in action-security.yml at %s", actionMetadata.GitHubToken.EnvironmentVariableName, filePath))
					return nil
				}
				if !reflect.DeepEqual(actionMetadata.GitHubToken.ActionInput, ActionInput{}) {
					lintIssues = append(lintIssues, fmt.Sprintf("Action input must not exist when there is no permissions. It is currently set to '%s' in action-security.yml at %s", actionMetadata.GitHubToken.ActionInput.Input, filePath))
					return nil
				}
			}

			//If permissions exist, Either Env variable or Action input must exist(not both)
			if len(actionMetadata.GitHubToken.Permissions.Scopes) != 0 {
				if actionMetadata.GitHubToken.EnvironmentVariableName != "" && !reflect.DeepEqual(actionMetadata.GitHubToken.ActionInput, ActionInput{}) {
					lintIssues = append(lintIssues, fmt.Sprintf("Either Environment variable or Action input should exist, both exist in action-security.yml at %s", filePath))
					return nil
				}
				if actionMetadata.GitHubToken.EnvironmentVariableName == "" && reflect.DeepEqual(actionMetadata.GitHubToken.ActionInput, ActionInput{}) {
					lintIssues = append(lintIssues, fmt.Sprintf("Either Environment variable or Action input should exist, none exist in action-security.yml at %s", filePath))
					return nil
				}
			}

			validScopes := []string{"actions", "checks", "contents", "deployments", "id-token", "issues", "packages",
				"pull-requests", "repository-projects", "security-events", "statuses"}
			mapScopes := make(map[string]bool)

			for _, scope := range validScopes {
				mapScopes[scope] = true
			}

			for key, scope := range actionMetadata.GitHubToken.Permissions.Scopes {

				_, found := mapScopes[key]
				if !found {
					lintIssues = append(lintIssues, fmt.Sprintf("Scope must be one of %s. It is currently %s in action-security.yml at %s", strings.Join(validScopes, ","), key, filePath))
					return nil
				}

				if scope.Permission != "read" && scope.Permission != "write" {
					lintIssues = append(lintIssues, fmt.Sprintf("Permissions must be either read or write. It is currently %s in action-security.yml at %s", scope.Permission, filePath))
					return nil
				}

				if !strings.HasPrefix(scope.Reason, "to ") {
					lintIssues = append(lintIssues, fmt.Sprintf("Reason must start with 'to '. It is currently %s in action-security.yml at %s", scope.Reason, filePath))
					return nil
				}

				if strings.HasSuffix(scope.Reason, ".") {
					lintIssues = append(lintIssues, fmt.Sprintf("Reason must not end with '.'. It is currently %s in action-security.yml at %s", scope.Reason, filePath))
					return nil
				}

				//Since the reason is added as a comment in the workflow file, limit the length to 50 to not clutter the workflow file
				if len(scope.Reason) > 50 {
					lintIssues = append(lintIssues, fmt.Sprintf("Reason must not exceed 50 char limit. It is currently %d in action-security.yml at %s", len(scope.Reason), filePath))
				}
			}

			return nil
		})
	if err != nil {
		log.Println(err)
	}

	if len(lintIssues) > 0 {
		for _, issue := range lintIssues {
			log.Println(issue)
		}
		t.Fail()
	}
}

func doesActionRepoExist(filePath string) bool {
	splitOnSlash := strings.Split(filePath, "/")
	owner := splitOnSlash[5]
	repo := splitOnSlash[6]

	PAT := os.Getenv("PAT")
	if len(PAT) == 0 {
		log.Println("doesActionRepoExist: PAT not set, skipping")
		return true
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: PAT},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	repository, _, err := client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		log.Println(fmt.Sprintf("error in doesActionRepoExist: %v", err))
		return false
	}
	branch := repository.DefaultBranch
	var ref github.RepositoryContentGetOptions
	ref.Ref = *branch

	// does the path to folder is correct for action repository
	if len(splitOnSlash) > 8 {
		folder := strings.Join(splitOnSlash[7:len(splitOnSlash)-1], "/")
		folder += "/action.yml"
		_, _, _, err = client.Repositories.GetContents(context.Background(), owner, repo, folder, &ref)

		if err != nil {
			folder := strings.Join(splitOnSlash[7:len(splitOnSlash)-1], "/")
			folder += "/action.yaml" // try out .yaml extension as well
			_, _, _, err = client.Repositories.GetContents(context.Background(), owner, repo, folder, &ref)

			if err != nil {
				log.Println(fmt.Sprintf("error in doesActionRepoExist: %v", err))
				return false
			}
		}
	}
	return true
}
