package main

import (
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const (
	ActionPermissionsTable = "ActionPermissions"
	MissingActionsTable    = "MissingActions"
)

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

func StoreActionPermissions(request string, svc dynamodbiface.DynamoDBAPI) error {

	var action Action

	err := json.Unmarshal([]byte(request), &action)

	if err != nil {
		return err
	}

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
