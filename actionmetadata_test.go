package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

func TestKnowledgeBase(t *testing.T) {
	kbFolder := os.Getenv("KBFolder")

	if kbFolder == "" {
		kbFolder = "knowledge-base"
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
			if info.Name() != "action-security.yml" {
				lintIssues = append(lintIssues, fmt.Sprintf("File must be named action-security.yml, not %s at %s", info.Name(), filePath))
				return nil
			}

			// validating the action repo
			var response *github.Response = doesActionRepoExist(filePath)
			if response.Response.StatusCode != http.StatusOK {
				lintIssues = append(lintIssues, fmt.Sprintf("Action repo does not exist at %s", filePath))
				return nil
			}
			defer response.Body.Close()

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

func doesActionRepoExist(filePath string) *github.Response {
	var Branch string = "master"
	splitOnSlash := strings.Split(filePath, "/")
	owner := splitOnSlash[1]
	repo := splitOnSlash[2]

	PAT := os.Getenv("PAT")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: PAT},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	_, res, err := client.Git.GetRef(context.Background(), owner, repo, fmt.Sprintf("tags/%s", Branch))

	if err != nil {
		_, res, err = client.Git.GetRef(context.Background(), owner, repo, fmt.Sprintf("heads/%s", Branch))
		if err != nil {
			return res
		}
	}
	return res
}
