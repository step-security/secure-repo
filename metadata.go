package main

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const ActionPermissionsTable = "ActionPermissions"

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
