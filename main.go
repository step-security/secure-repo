package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type Handler struct {
}

func (h Handler) Invoke(ctx context.Context, req []byte) ([]byte, error) {

	httpRequest := &events.APIGatewayV2HTTPRequest{}

	err := json.Unmarshal([]byte(req), &httpRequest)

	if err == nil && httpRequest.RawPath != "" {
		fmt.Printf("Request is APIGatewayV2HTTPRequest: %v\n", httpRequest)

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		dynamoDbSvc := dynamodb.New(sess)

		fixResponse, err := FixWorkflowPermissions(httpRequest.Body, dynamoDbSvc)

		var response events.APIGatewayProxyResponse

		if err != nil {

			response = events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       err.Error(),
			}
		} else {

			if len(fixResponse.MissingActions) > 0 {
				storeMissingActions(fixResponse.MissingActions, dynamoDbSvc)
			}

			output, _ := json.Marshal(fixResponse)
			response = events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Body:       string(output),
			}
		}

		returnValue, _ := json.Marshal(&response)

		return returnValue, nil

	}

	return nil, fmt.Errorf("request was neither APIGatewayV2HTTPRequest nor SQSEvent")
}

func main() {
	lambda.StartHandler(Handler{})
}
