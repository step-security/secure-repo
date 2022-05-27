package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type GitHubWorkflowSecrets struct {
	Repo          string   `json:"repo"`
	RunId         string   `json:"runId"`
	AreSecretsSet bool     `json:"areSecretsSet"`
	Secrets       []Secret `json:"secrets"`
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

func getWorkflowSecrets(queryStringParams map[string]string, svc dynamodbiface.DynamoDBAPI) (*GitHubWorkflowSecrets, error) {
	tableName := GitHubWorkflowSecretsTableName

	owner := queryStringParams["owner"]
	repo := queryStringParams["repo"]
	runId := queryStringParams["runId"]

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

func GetSecrets(queryStringParams map[string]string, svc dynamodbiface.DynamoDBAPI) (*GitHubWorkflowSecrets, error) {
	// Get the record for repo and run id
	gitHubWorkflowSecrets, err := getWorkflowSecrets(queryStringParams, svc)
	if err != nil {
		return nil, err
	}

	// If record exists, check if secrets are set
	if gitHubWorkflowSecrets != nil {
		return gitHubWorkflowSecrets, nil
	}

	// If record does not exist, insert record
	gitHubWorkflowSecrets = &GitHubWorkflowSecrets{}
	owner := queryStringParams["owner"]
	repo := queryStringParams["repo"]

	gitHubWorkflowSecrets.Repo = fmt.Sprintf("%s/%s", owner, repo)
	gitHubWorkflowSecrets.RunId = queryStringParams["runId"]
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
