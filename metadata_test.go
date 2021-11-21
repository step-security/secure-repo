package main

import (
	"io/ioutil"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
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

	actionPermissionsYaml, err := ioutil.ReadFile("./testfiles/action-permissions.yml")

	if err != nil {
		return nil, err
	}

	actionPermissions := ActionPermissions{}

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

func TestStoreActionPermissions(t *testing.T) {
	type args struct {
		actionName string
		request    string
		svc        dynamodbiface.DynamoDBAPI
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "no perms",
			args:    args{actionName: "owner/repo@v1", request: "{}", svc: &mockDynamoDBClient{}},
			wantErr: false,
		},
		{
			name:    "content read",
			args:    args{actionName: "owner/repo@v1", request: "{\"permissions\":{\"contents\":\"read\"}}", svc: &mockDynamoDBClient{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StoreActionPermissions(tt.args.actionName, tt.args.request, tt.args.svc); (err != nil) != tt.wantErr {
				t.Errorf("StoreActionPermissions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
