package main

import (
	"testing"
)

func TestInvokeHTTPRequest(t *testing.T) {
	/*sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create DynamoDB client
	dynamoDbSvc := dynamodb.New(sess)
	importActions(dynamoDbSvc)*/

	/*request := &events.APIGatewayV2HTTPRequest{}

	request.RawPath = "v1/secure-workflow"
	input, _ := ioutil.ReadFile(path.Join("./testfiles/input", "action-issues.yml"))
	request.Body = string(input)

	bytes, _ := json.Marshal(request)

	handler := &Handler{}
	response, err := handler.Invoke(context.TODO(), bytes)

	if err != nil {
		t.Errorf("error not expected %v", err)
	}

	proxyResponse := &events.APIGatewayProxyResponse{}
	err = json.Unmarshal(response, &proxyResponse)*/
}
