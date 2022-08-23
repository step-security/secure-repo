package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwk"
)

type GitHubWorkflowSecrets struct {
	Repo           string   `json:"repo"`
	RunId          string   `json:"runId"`
	AreSecretsSet  bool     `json:"areSecretsSet"`
	Secrets        []Secret `json:"secrets"`
	Ref            string   `json:"ref"`
	RefType        string   `json:"ref_type"`
	Workflow       string   `json:"workflow"`
	EventName      string   `json:"event_name"`
	JobWorkflowRef string   `json:"job_workflow_ref"`
}

type Secret struct {
	Name  string
	Value string
}

const (
	GitHubWorkflowSecretsTableName = "GitHubWorkflowSecrets"
	GitHubRunId                    = "runId"
	GitHubOwner                    = "owner"
	GitHubRepo                     = "repo"
)

func getWorkflowSecrets(owner, repo, runId string, svc dynamodbiface.DynamoDBAPI) (*GitHubWorkflowSecrets, error) {
	tableName := GitHubWorkflowSecretsTableName

	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			GitHubRepo: {
				S: aws.String(fmt.Sprintf("%s/%s", owner, repo)),
			},
			GitHubRunId: {
				S: aws.String(runId),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	item := GitHubWorkflowSecrets{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func setWorkflowSecrets(gitHubWorkflowSecrets GitHubWorkflowSecrets, dynamoDbSvc dynamodbiface.DynamoDBAPI) error {
	tableName := GitHubWorkflowSecretsTableName

	av, err := dynamodbattribute.MarshalMap(gitHubWorkflowSecrets)

	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}

	_, err = dynamoDbSvc.PutItem(input)
	if err != nil {
		return err
	}

	return nil
}

const jwksURL = `https://token.actions.githubusercontent.com/.well-known/jwks`

func getKey(token *jwt.Token) (interface{}, error) {

	// TODO: cache response so we don't have to make a request every time
	// we want to verify a JWT
	set, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		return nil, err
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have string kid")
	}

	key, match := set.LookupKeyID(keyID)
	if match {

		var rawkey interface{} // This is the raw key, like *rsa.PrivateKey or *ecdsa.PrivateKey
		if err := key.Raw(&rawkey); err != nil {
			return nil, fmt.Errorf("failed to create public key")
		}
		return rawkey, nil
	}

	return nil, fmt.Errorf("no key found in jwksURL")
}

func getClaimsFromAuthToken(authHeader string, skipClaimValidation bool) (*GitHubWorkflowSecrets, error) {
	gitHubWorkflowSecrets := GitHubWorkflowSecrets{}
	tokenParts := strings.Split(authHeader, "Bearer ")
	if len(tokenParts) < 2 {
		return nil, fmt.Errorf("token not set with Bearer keyword")
	}
	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = skipClaimValidation
	token, err := parser.Parse(tokenParts[1], getKey)
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(jwt.MapClaims)
	gitHubWorkflowSecrets.Repo = claims["repository"].(string)
	gitHubWorkflowSecrets.RunId = claims["run_id"].(string)
	gitHubWorkflowSecrets.Workflow = claims["workflow"].(string)
	gitHubWorkflowSecrets.EventName = claims["event_name"].(string)
	gitHubWorkflowSecrets.Ref = claims["ref"].(string)
	gitHubWorkflowSecrets.RefType = claims["ref_type"].(string)
	gitHubWorkflowSecrets.JobWorkflowRef = claims["job_workflow_ref"].(string)
	return &gitHubWorkflowSecrets, nil
}

func GetSecrets(queryStringParams map[string]string, authHeader string, svc dynamodbiface.DynamoDBAPI) (*GitHubWorkflowSecrets, error) {
	owner := ""
	repo := ""
	runId := ""
	var err error
	authHeaderVerified := false
	var gitHubWorkflowSecrets *GitHubWorkflowSecrets
	// this is a call from the GitHub Action
	if len(authHeader) > 0 {
		// verify OIDC token
		gitHubWorkflowSecrets, err = getClaimsFromAuthToken(authHeader, svc == nil) // skip validation for unit tests
		if err != nil {
			return nil, err
		}

		repositoryParts := strings.Split(gitHubWorkflowSecrets.Repo, "/")
		if len(repositoryParts) == 2 {
			owner = repositoryParts[0]
			repo = repositoryParts[1]
		}

		authHeaderVerified = true
	} else {
		owner = queryStringParams["owner"]
		repo = queryStringParams["repo"]
		runId = queryStringParams["runId"]
	}

	// Get the record for repo and run id
	gitHubWorkflowSecrets, err = getWorkflowSecrets(owner, repo, runId, svc)
	if err != nil {
		return nil, err
	}

	// If record exists, check if secrets are set
	if gitHubWorkflowSecrets != nil {
		if !authHeaderVerified && gitHubWorkflowSecrets.AreSecretsSet {
			return nil, fmt.Errorf("once secrets are set, they can only be returned to GitHub workflow")
		}
		return gitHubWorkflowSecrets, nil
	}

	// If record does not exist, insert record
	gitHubWorkflowSecrets.AreSecretsSet = false
	secrets := strings.Split(queryStringParams["secrets"], ",")
	for _, secret := range secrets {
		gitHubWorkflowSecrets.Secrets = append(gitHubWorkflowSecrets.Secrets, Secret{Name: secret, Value: ""})
	}

	err = setWorkflowSecrets(*gitHubWorkflowSecrets, svc)

	if err != nil {
		return nil, err
	}

	return gitHubWorkflowSecrets, nil
}

func DeleteSecrets(authHeader string, svc dynamodbiface.DynamoDBAPI) error {
	owner := ""
	repo := ""
	var err error
	var gitHubWorkflowSecrets *GitHubWorkflowSecrets
	// this is a call from the GitHub Action
	if len(authHeader) > 0 {
		// verify OIDC token
		gitHubWorkflowSecrets, err = getClaimsFromAuthToken(authHeader, svc == nil) // skip validation for unit tests
		if err != nil {
			return err
		}

	} else {
		return fmt.Errorf("only GitHub workflow can delete the secrets")
	}
	repositoryParts := strings.Split(gitHubWorkflowSecrets.Repo, "/")
	if len(repositoryParts) == 2 {
		owner = repositoryParts[0]
		repo = repositoryParts[1]
	}

	// Get the record for repo and run id
	gitHubWorkflowSecrets, err = getWorkflowSecrets(owner, repo, gitHubWorkflowSecrets.RunId, svc)
	if err != nil {
		return err
	}

	// If record exists, check if secrets are set
	if gitHubWorkflowSecrets != nil {
		if gitHubWorkflowSecrets.AreSecretsSet {

			for _, secret := range gitHubWorkflowSecrets.Secrets {
				secret.Value = ""
			}

			err = setWorkflowSecrets(*gitHubWorkflowSecrets, svc)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func SetSecrets(body string, svc dynamodbiface.DynamoDBAPI) error {
	var gitHubWorkflowSecrets GitHubWorkflowSecrets

	err := json.Unmarshal([]byte(body), &gitHubWorkflowSecrets)

	if err != nil {
		return err
	}

	err = setWorkflowSecrets(gitHubWorkflowSecrets, svc)

	if err != nil {
		return err
	}

	return nil
}
