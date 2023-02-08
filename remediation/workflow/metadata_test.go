package workflow

import (
	"io/ioutil"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	metadata "github.com/step-security/secure-repo/remediation/workflow/metadata"
	"gopkg.in/yaml.v3"
)

// Define a mock struct to be used in your unit tests of myFunc.
type mockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
}

func (m *mockDynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	// mock response/functionality
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDynamoDBClient) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {

	actionPermissionsYaml, err := ioutil.ReadFile("../../testfiles/action-permissions.yml")

	if err != nil {
		return nil, err
	}

	actionPermissions := metadata.ActionPermissions{}

	err = yaml.Unmarshal(actionPermissionsYaml, &actionPermissions)

	if err != nil {
		return nil, err
	}

	output := &dynamodb.ScanOutput{}

	for _, d := range actionPermissions.Actions {
		av, _ := dynamodbattribute.MarshalMap(d)
		output.Items = append(output.Items, av)
	}

	return output, nil

}
