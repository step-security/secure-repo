package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"gopkg.in/yaml.v3"
)

const (
	ActionPermissionsTable = "ActionPermissions"
	MissingActionsTable    = "MissingActions"
)

func getActionPermissions(svc dynamodbiface.DynamoDBAPI) (*ActionPermissions, error) {

	actionPermissions := ActionPermissions{}
	actionPermissions.Actions = make(map[string]Action)

	tableName := ActionPermissionsTable

	params := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	// Make the DynamoDB Query API call
	result, err := svc.Scan(params)
	if err != nil {
		return nil, fmt.Errorf("query API call failed: %s", err.Error())
	}

	for _, i := range result.Items {
		item := Action{}

		err = dynamodbattribute.UnmarshalMap(i, &item)

		if err != nil {
			return nil, fmt.Errorf("got error unmarshalling: %s", err.Error())
		}

		actionKey := strings.ReplaceAll(item.Name, "/", "-")
		actionKey = strings.ToLower(actionKey)

		actionPermissions.Actions[actionKey] = item

	}

	return &actionPermissions, nil

}

func importActions(svc dynamodbiface.DynamoDBAPI) {
	actionPermissionsYaml, err := ioutil.ReadFile("./testfiles/action-permissions.yml")

	if err != nil {
		return
	}

	actionPermissions := ActionPermissions{}

	err = yaml.Unmarshal(actionPermissionsYaml, &actionPermissions)

	if err != nil {
		return
	}

	for _, action := range actionPermissions.Actions {

		av, err := dynamodbattribute.MarshalMap(action)

		if err != nil {
			return
		}

		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(ActionPermissionsTable),
		}

		_, err = svc.PutItem(input)
		if err != nil {
			return
		}
	}
}

func StoreMissingActions(missingActions []string, svc dynamodbiface.DynamoDBAPI) error {

	for _, action := range missingActions {

		atIndex := strings.Index(action, "@")

		if atIndex == -1 {
			continue
		}

		actionKey := action[0:atIndex]

		input := dynamodb.PutItemInput{
			TableName: aws.String(MissingActionsTable),
			Item: map[string]*dynamodb.AttributeValue{
				"Name": {
					S: aws.String(actionKey),
				},
			},
		}

		_, err := svc.PutItem(&input)
		if err != nil {
			return err
		}

	}

	return nil
}

func StoreActionPermissions(actionName, request string, svc dynamodbiface.DynamoDBAPI) error {

	var action Action

	err := json.Unmarshal([]byte(request), &action)

	if err != nil {
		return err
	}

	atIndex := strings.Index(actionName, "@")
	actionKey := actionName[0:atIndex]
	actionKey = strings.ReplaceAll(actionKey, "/", "-")
	actionKey = strings.ToLower(actionKey)
	action.Name = actionKey

	av, err := dynamodbattribute.MarshalMap(action)

	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(ActionPermissionsTable),
	}

	_, err = svc.PutItem(input)
	if err != nil {
		return err
	}

	return nil
}
