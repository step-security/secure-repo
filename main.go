package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/step-security/secure-repo/remediation/dependabot"
	"github.com/step-security/secure-repo/remediation/docker"
	"github.com/step-security/secure-repo/remediation/secrets"
	"github.com/step-security/secure-repo/remediation/workflow"
	"github.com/step-security/secure-repo/remediation/workflow/permissions"
)

type Handler struct {
}

func (h Handler) Invoke(ctx context.Context, req []byte) ([]byte, error) {

	httpRequest := &events.APIGatewayV2HTTPRequest{}

	err := json.Unmarshal([]byte(req), &httpRequest)

	if err == nil && httpRequest.RawPath != "" {

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		dynamoDbSvc := dynamodb.New(sess)
		var response events.APIGatewayProxyResponse

		if httpRequest.RequestContext.HTTP.Method == "OPTIONS" {
			response = events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
			}
			returnValue, _ := json.Marshal(&response)
			return returnValue, nil
		}

		if strings.Contains(httpRequest.RawPath, "/secrets") {
			if httpRequest.RequestContext.HTTP.Method == "GET" {
				authHeader := httpRequest.Headers["authorization"]
				githubWorkflowSecrets, err := secrets.GetSecrets(httpRequest.QueryStringParameters, authHeader, dynamoDbSvc, false)
				if err != nil {
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusInternalServerError,
						Body:       err.Error(),
					}
				} else {
					output, _ := json.Marshal(githubWorkflowSecrets)
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusOK,
						Body:       string(output),
					}
				}

			} else if httpRequest.RequestContext.HTTP.Method == "PUT" {
				authHeader := httpRequest.Headers["authorization"]
				githubWorkflowSecrets, err := secrets.InitSecrets(httpRequest.Body, authHeader, dynamoDbSvc)
				if err != nil {
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusInternalServerError,
						Body:       err.Error(),
					}
				} else {
					output, _ := json.Marshal(githubWorkflowSecrets)
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusOK,
						Body:       string(output),
					}
				}

			} else if httpRequest.RequestContext.HTTP.Method == "POST" {
				err := secrets.SetSecrets(httpRequest.Body, dynamoDbSvc)
				if err != nil {
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusInternalServerError,
						Body:       err.Error(),
					}
				} else {
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusOK,
					}
				}
			} else if httpRequest.RequestContext.HTTP.Method == "DELETE" {
				authHeader := httpRequest.Headers["authorization"]
				err := secrets.DeleteSecrets(authHeader, dynamoDbSvc)
				if err != nil {
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusInternalServerError,
						Body:       err.Error(),
					}
				} else {
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusOK,
					}
				}
			}
		}

		if strings.Contains(httpRequest.RawPath, "/secure-workflow") {

			inputYaml := ""
			queryStringParams := httpRequest.QueryStringParameters
			// if owner is set, assuming that repo, path are also set
			// get the workflow using API
			if _, ok := queryStringParams["owner"]; ok {
				inputYaml, err = workflow.GetGitHubWorkflowContents(httpRequest.QueryStringParameters)
				if err != nil {
					fixResponse := &permissions.SecureWorkflowReponse{WorkflowFetchError: true, HasErrors: true}
					output, _ := json.Marshal(fixResponse)
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusOK,
						Body:       string(output),
					}
					returnValue, _ := json.Marshal(&response)
					return returnValue, nil
				}
			} else {
				// if owner is not set, then workflow should be sent in the body
				inputYaml = httpRequest.Body
			}

			fixResponse, err := workflow.SecureWorkflow(httpRequest.QueryStringParameters, inputYaml, dynamoDbSvc)

			if err != nil {
				response = events.APIGatewayProxyResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       err.Error(),
				}
			} else {

				output, _ := json.Marshal(fixResponse)
				response = events.APIGatewayProxyResponse{
					StatusCode: http.StatusOK,
					Body:       string(output),
				}
			}

		}

		if strings.Contains(httpRequest.RawPath, "/secure-dockerfile") {

			dockerFile := ""
			queryStringParams := httpRequest.QueryStringParameters
			// if owner is set, assuming that repo, path are also set
			// get the dockerfile using API
			if _, ok := queryStringParams["owner"]; ok {
				dockerFile, err = workflow.GetGitHubWorkflowContents(httpRequest.QueryStringParameters)
				if err != nil {
					fixResponse := &docker.SecureDockerfileResponse{DockerfileFetchError: true}
					output, _ := json.Marshal(fixResponse)
					response = events.APIGatewayProxyResponse{
						StatusCode: http.StatusOK,
						Body:       string(output),
					}
					returnValue, _ := json.Marshal(&response)
					return returnValue, nil
				}
			} else {
				// if owner is not set, then dockerfile should be sent in the body
				dockerFile = httpRequest.Body
			}

			fixResponse, err := docker.SecureDockerFile(dockerFile)
			if err != nil {
				response = events.APIGatewayProxyResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       err.Error(),
				}
			} else {

				output, _ := json.Marshal(fixResponse)
				response = events.APIGatewayProxyResponse{
					StatusCode: http.StatusOK,
					Body:       string(output),
				}
			}

		}

		if strings.Contains(httpRequest.RawPath, "/update-dependabot-config") {

			updateDependabotConfigRequest := ""
			updateDependabotConfigRequest = httpRequest.Body

			fixResponse, err := dependabot.UpdateDependabotConfig(updateDependabotConfigRequest)
			if err != nil {
				response = events.APIGatewayProxyResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       err.Error(),
				}
			} else {

				output, _ := json.Marshal(fixResponse)
				response = events.APIGatewayProxyResponse{
					StatusCode: http.StatusOK,
					Body:       string(output),
				}
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
