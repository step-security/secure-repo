package secrets

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/jarcoal/httpmock"
)

type mockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
}

func (m *mockDynamoDBClient) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	output := &dynamodb.GetItemOutput{}
	gitHubWorkflowSecrets := &GitHubWorkflowSecrets{Repo: "varunsh-coder/actions-playground", RunId: "2800694956",
		AreSecretsSet: true, Secrets: []Secret{{Name: "test", Value: "123"}}}

	av, _ := dynamodbattribute.MarshalMap(gitHubWorkflowSecrets)
	output.Item = av
	return output, nil
}
func Test_getClaimsFromAuthToken(t *testing.T) {
	type args struct {
		authHeader          string
		skipClaimValidation bool
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://token.actions.githubusercontent.com/.well-known/jwks",
		httpmock.NewStringResponder(200, `{"keys":[{"n":"4WpHpoBYsVBVfSlfgnRbdPMxP3Eb7rFqE48e4pPM4qH_9EsUZIi21LjOu8UkKn14L4hrRfzfRHG7VQSbxXBU1Qa-xM5yVxdmfQZKBxQnPWaE1v7edjxq1ZYnqHIp90Uvnw6798xMCSvI_V3FR8tix5GaoTgkixXlPc-ozifMyEZMmhvuhfDsSxQeTSHGPlWfGkX0id_gYzKPeI69EGtQ9ZN3PLTdoAI8jxlQ-jyDchi9h2ax6hgMLDsMZyiIXnF2UYq4j36Cs5RgdC296d0hEOHN0WYZE-xPl7y_A9UHcVjrxeGfVOuTBXqjowofimn4ESnVXNReCsOwZCJlvJzfpQ","kty":"RSA","kid":"78167F727DEC5D801DD1C8784C704A1C880EC0E1","alg":"RS256","e":"AQAB","use":"sig","x5c":["MIIDrDCCApSgAwIBAgIQMPdKi0TFTMqmg1HHo6FfsDANBgkqhkiG9w0BAQsFADA2MTQwMgYDVQQDEyt2c3RzLXZzdHNnaHJ0LWdoLXZzby1vYXV0aC52aXN1YWxzdHVkaW8uY29tMB4XDTIyMDEwNTE4NDcyMloXDTI0MDEwNTE4NTcyMlowNjE0MDIGA1UEAxMrdnN0cy12c3RzZ2hydC1naC12c28tb2F1dGgudmlzdWFsc3R1ZGlvLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAOFqR6aAWLFQVX0pX4J0W3TzMT9xG+6xahOPHuKTzOKh//RLFGSIttS4zrvFJCp9eC+Ia0X830Rxu1UEm8VwVNUGvsTOclcXZn0GSgcUJz1mhNb+3nY8atWWJ6hyKfdFL58Ou/fMTAkryP1dxUfLYseRmqE4JIsV5T3PqM4nzMhGTJob7oXw7EsUHk0hxj5VnxpF9Inf4GMyj3iOvRBrUPWTdzy03aACPI8ZUPo8g3IYvYdmseoYDCw7DGcoiF5xdlGKuI9+grOUYHQtvendIRDhzdFmGRPsT5e8vwPVB3FY68Xhn1TrkwV6o6MKH4pp+BEp1VzUXgrDsGQiZbyc36UCAwEAAaOBtTCBsjAOBgNVHQ8BAf8EBAMCBaAwCQYDVR0TBAIwADAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwNgYDVR0RBC8wLYIrdnN0cy12c3RzZ2hydC1naC12c28tb2F1dGgudmlzdWFsc3R1ZGlvLmNvbTAfBgNVHSMEGDAWgBRZBaZCR9ghvStfcWaGwuHGjrfTgzAdBgNVHQ4EFgQUWQWmQkfYIb0rX3FmhsLhxo6304MwDQYJKoZIhvcNAQELBQADggEBAGNdfALe6mdxQ67QL8GlW4dfFwvCX87JOeZThZ9uCj1+x1xUnywoR4o5q2DVI/JCvBRPn0BUb3dEVWLECXDHGjblesWZGMdSGYhMzWRQjVNmCYBC1ZM5QvonWCBcGkd72mZx0eFHnJCAP/TqEEpRvMHR+OOtSiZWV9zZpF1tf06AjKwT64F9V8PCmSIqPJXcTQXKKfkHZmGUk9AYF875+/FfzF89tCnT53UEh5BldFz0SAls+NhexbW/oOokBNCVqe+T2xXizktbFnFAFaomvwjVSvIeu3i/0Ygywl+3s5izMEsZ1T1ydIytv4FZf2JCHgRpmGPWJ5A7TpxuHSiE8Do="],"x5t":"eBZ_cn3sXYAd0ch4THBKHIgOwOE"},{"n":"wgCsNL8S6evSH_AHBsps2ccIHSwLpuEUGS9GYenGmGkSKyWefKsZheKl_84voiUgduuKcKA2aWQezp9338LjtlBmTHjopzAeU-Q3_IvqNf7BfrEAzEyp-ymdhNzPTE7Snmr5o_9AeiP1ZDBo35FaULgVUECJ3AzAM36zkURax3VNZRRZx1gb8lPUs9M5Yw6aZpHSOd6q_QzE8CP1OhGrAdoBzZ6ZCElon0kI-IuRLCwKptS7Yroi5-RtEKD2W458axNAQ36Yw93N8kInUC1QZDPrKd4QfYiG68ywjBoxp_bjNg5kh4LJmq1mwyGdNQV6F1Ew_jYlmou2Y8wvHQRJPQ","kty":"RSA","kid":"52F197C481DE70112C441B4A9B37B53C7FCF0DB5","alg":"RS256","e":"AQAB","use":"sig","x5c":["MIIDrDCCApSgAwIBAgIQLQnoXJ3HT6uPYvEofvOZ6zANBgkqhkiG9w0BAQsFADA2MTQwMgYDVQQDEyt2c3RzLXZzdHNnaHJ0LWdoLXZzby1vYXV0aC52aXN1YWxzdHVkaW8uY29tMB4XDTIxMTIwNjE5MDUyMloXDTIzMTIwNjE5MTUyMlowNjE0MDIGA1UEAxMrdnN0cy12c3RzZ2hydC1naC12c28tb2F1dGgudmlzdWFsc3R1ZGlvLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMIArDS/Eunr0h/wBwbKbNnHCB0sC6bhFBkvRmHpxphpEislnnyrGYXipf/OL6IlIHbrinCgNmlkHs6fd9/C47ZQZkx46KcwHlPkN/yL6jX+wX6xAMxMqfspnYTcz0xO0p5q+aP/QHoj9WQwaN+RWlC4FVBAidwMwDN+s5FEWsd1TWUUWcdYG/JT1LPTOWMOmmaR0jneqv0MxPAj9ToRqwHaAc2emQhJaJ9JCPiLkSwsCqbUu2K6IufkbRCg9luOfGsTQEN+mMPdzfJCJ1AtUGQz6yneEH2IhuvMsIwaMaf24zYOZIeCyZqtZsMhnTUFehdRMP42JZqLtmPMLx0EST0CAwEAAaOBtTCBsjAOBgNVHQ8BAf8EBAMCBaAwCQYDVR0TBAIwADAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwNgYDVR0RBC8wLYIrdnN0cy12c3RzZ2hydC1naC12c28tb2F1dGgudmlzdWFsc3R1ZGlvLmNvbTAfBgNVHSMEGDAWgBTTNQQWmG4PZZsdfMeamCH1YcyDZTAdBgNVHQ4EFgQU0zUEFphuD2WbHXzHmpgh9WHMg2UwDQYJKoZIhvcNAQELBQADggEBAK/d+HzBSRac7p6CTEolRXcBrBmmeJUDbBy20/XA6/lmKq73dgc/za5VA6Kpfd6EFmG119tl2rVGBMkQwRx8Ksr62JxmCw3DaEhE8ZjRARhzgSiljqXHlk8TbNnKswHxWmi4MD2/8QhHJwFj3X35RrdMM4R0dN/ojLlWsY9jXMOAvcSBQPBqttn/BjNzvn93GDrVafyX9CPl8wH40MuWS/gZtXeYIQg5geQkHCyP96M5Sy8ZABOo9MSIfPRw1F7dqzVuvliul9ZZGV2LsxmZCBtbsCkBau0amerigZjud8e9SNp0gaJ6wGhLbstCZIdaAzS5mSHVDceQzLrX2oe1h4k="],"x5t":"UvGXxIHecBEsRBtKmze1PH_PDbU"}]}`))

	tests := []struct {
		name    string
		args    args
		want    *GitHubWorkflowSecrets
		wantErr bool
	}{
		{name: "pass_expected", args: args{authHeader: "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6ImVCWl9jbjNzWFlBZDBjaDRUSEJLSElnT3dPRSIsImtpZCI6Ijc4MTY3RjcyN0RFQzVEODAxREQxQzg3ODRDNzA0QTFDODgwRUMwRTEifQ.eyJqdGkiOiIzNGI0YTM1Ny05MjQ1LTRmYjgtOTlmYy00NDc3OWRiM2NmMzkiLCJzdWIiOiJyZXBvOnZhcnVuc2gtY29kZXIvYWN0aW9ucy1wbGF5Z3JvdW5kOnJlZjpyZWZzL2hlYWRzL21haW4iLCJhdWQiOiJodHRwczovL2dpdGh1Yi5jb20vdmFydW5zaC1jb2RlciIsInJlZiI6InJlZnMvaGVhZHMvbWFpbiIsInNoYSI6IjUzNmNmN2IwNGVlZGEyYmQ2ZGFmOTBjNTgzM2Q5ZTkwYjk1MTkyNGUiLCJyZXBvc2l0b3J5IjoidmFydW5zaC1jb2Rlci9hY3Rpb25zLXBsYXlncm91bmQiLCJyZXBvc2l0b3J5X293bmVyIjoidmFydW5zaC1jb2RlciIsInJlcG9zaXRvcnlfb3duZXJfaWQiOiIyNTAxNTkxNyIsInJ1bl9pZCI6IjI4MDA2OTQ5NTYiLCJydW5fbnVtYmVyIjoiNiIsInJ1bl9hdHRlbXB0IjoiMSIsInJlcG9zaXRvcnlfdmlzaWJpbGl0eSI6InB1YmxpYyIsInJlcG9zaXRvcnlfaWQiOiI0MzM5MDM3OTIiLCJhY3Rvcl9pZCI6IjI1MDE1OTE3IiwiYWN0b3IiOiJ2YXJ1bnNoLWNvZGVyIiwid29ya2Zsb3ciOiJQdWJsaXNoIFBhY2thZ2UgdG8gbnBtanMiLCJoZWFkX3JlZiI6IiIsImJhc2VfcmVmIjoiIiwiZXZlbnRfbmFtZSI6IndvcmtmbG93X2Rpc3BhdGNoIiwicmVmX3R5cGUiOiJicmFuY2giLCJqb2Jfd29ya2Zsb3dfcmVmIjoidmFydW5zaC1jb2Rlci9hY3Rpb25zLXBsYXlncm91bmQvLmdpdGh1Yi93b3JrZmxvd3MvbWZhX3JlbGVhc2UueW1sQHJlZnMvaGVhZHMvbWFpbiIsImlzcyI6Imh0dHBzOi8vdG9rZW4uYWN0aW9ucy5naXRodWJ1c2VyY29udGVudC5jb20iLCJuYmYiOjE2NTk2NjIzOTMsImV4cCI6MTY1OTY2MzI5MywiaWF0IjoxNjU5NjYyOTkzfQ.O-SRv44w8cHSsvQ40ntM5yqXTx4xLnp3koHZVwNcnes2DPGzbcXbf_qzmJqwpSVBqBjQUDS-nKLD_NgM8XSSgIQiTTIL0CBgZCb2FAwkYaVFWoMR38F1Z2OvHKz_WgsvaTX9thfMHyTe3gbFr1B8JSv2MeBQbFODCw7F1mkIPGPCd5wVAKjY3ECZp2JCmQ8nNvMtZj-HvuK5g3bXRpZASePufjhN2MP2y_ewGydWyNYIT6_sNIw8pab4eeD7VEaCTaxq4_yQkayPr49_xB5-g8H6LvY_aLMczJq9NpQMboEfFtlnQVQ90g4F7bFQd_cdMZPquKT0AJmDEsu04F1Hag", skipClaimValidation: true},
			want: &GitHubWorkflowSecrets{Repo: "varunsh-coder/actions-playground", RunId: "2800694956", Ref: "refs/heads/main",
				RefType: "branch", Workflow: "Publish Package to npmjs", EventName: "workflow_dispatch", JobWorkflowRef: "varunsh-coder/actions-playground/.github/workflows/mfa_release.yml@refs/heads/main"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getClaimsFromAuthToken(tt.args.authHeader, tt.args.skipClaimValidation)
			if (err != nil) != tt.wantErr {
				t.Errorf("getClaimsFromAuthToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getClaimsFromAuthToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSecretsFromString(t *testing.T) {
	type args struct {
		body string
	}
	tests := []struct {
		name    string
		args    args
		want    []Secret
		wantErr bool
	}{
		{name: "single secret no name no description", args: args{body: `["AWS_ACCESS_KEY_ID:"]`}, want: []Secret{{Name: "AWS_ACCESS_KEY_ID"}}, wantErr: false},
		{name: "single secret name no description", args: args{body: `["AWS_ACCESS_KEY_ID:", "name: aws access key id"]`}, want: []Secret{{Name: "AWS_ACCESS_KEY_ID", SecretName: "aws access key id"}}, wantErr: false},
		{name: "single secret name description", args: args{body: `["AWS_ACCESS_KEY_ID:", "name: aws access key id", "description: aws access key id for prod"]`}, want: []Secret{{Name: "AWS_ACCESS_KEY_ID", SecretName: "aws access key id", Description: "aws access key id for prod"}}, wantErr: false},
		{name: "multi secret name description", args: args{body: `["AWS_ACCESS_KEY_ID:", "name: aws access key id", "description: aws access key id for prod", "AWS_SECRET_ACCESS_KEY:", "name: AWS secret access key", "description: this is the secret"]`},
			want: []Secret{{Name: "AWS_ACCESS_KEY_ID", SecretName: "aws access key id", Description: "aws access key id for prod"},
				{Name: "AWS_SECRET_ACCESS_KEY", SecretName: "AWS secret access key", Description: "this is the secret"}}, wantErr: false},
		{name: "multi secret with space name description", args: args{body: `["AWS_ACCESS_KEY_ID: ","  name: 'AWS access key'","  description: 'this is the access key'","AWS_SECRET_ACCESS_KEY:","  name: 'AWS secret access key'","  description: 'this is the secret'"]`},
			want: []Secret{{Name: "AWS_ACCESS_KEY_ID", SecretName: "'AWS access key'", Description: "'this is the access key'"},
				{Name: "AWS_SECRET_ACCESS_KEY", SecretName: "'AWS secret access key'", Description: "'this is the secret'"}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSecretsFromString(tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSecretsFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSecretsFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSecrets(t *testing.T) {
	type args struct {
		queryStringParams map[string]string
		authHeader        string
		svc               dynamodbiface.DynamoDBAPI
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://token.actions.githubusercontent.com/.well-known/jwks",
		httpmock.NewStringResponder(200, `{"keys":[{"n":"4WpHpoBYsVBVfSlfgnRbdPMxP3Eb7rFqE48e4pPM4qH_9EsUZIi21LjOu8UkKn14L4hrRfzfRHG7VQSbxXBU1Qa-xM5yVxdmfQZKBxQnPWaE1v7edjxq1ZYnqHIp90Uvnw6798xMCSvI_V3FR8tix5GaoTgkixXlPc-ozifMyEZMmhvuhfDsSxQeTSHGPlWfGkX0id_gYzKPeI69EGtQ9ZN3PLTdoAI8jxlQ-jyDchi9h2ax6hgMLDsMZyiIXnF2UYq4j36Cs5RgdC296d0hEOHN0WYZE-xPl7y_A9UHcVjrxeGfVOuTBXqjowofimn4ESnVXNReCsOwZCJlvJzfpQ","kty":"RSA","kid":"78167F727DEC5D801DD1C8784C704A1C880EC0E1","alg":"RS256","e":"AQAB","use":"sig","x5c":["MIIDrDCCApSgAwIBAgIQMPdKi0TFTMqmg1HHo6FfsDANBgkqhkiG9w0BAQsFADA2MTQwMgYDVQQDEyt2c3RzLXZzdHNnaHJ0LWdoLXZzby1vYXV0aC52aXN1YWxzdHVkaW8uY29tMB4XDTIyMDEwNTE4NDcyMloXDTI0MDEwNTE4NTcyMlowNjE0MDIGA1UEAxMrdnN0cy12c3RzZ2hydC1naC12c28tb2F1dGgudmlzdWFsc3R1ZGlvLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAOFqR6aAWLFQVX0pX4J0W3TzMT9xG+6xahOPHuKTzOKh//RLFGSIttS4zrvFJCp9eC+Ia0X830Rxu1UEm8VwVNUGvsTOclcXZn0GSgcUJz1mhNb+3nY8atWWJ6hyKfdFL58Ou/fMTAkryP1dxUfLYseRmqE4JIsV5T3PqM4nzMhGTJob7oXw7EsUHk0hxj5VnxpF9Inf4GMyj3iOvRBrUPWTdzy03aACPI8ZUPo8g3IYvYdmseoYDCw7DGcoiF5xdlGKuI9+grOUYHQtvendIRDhzdFmGRPsT5e8vwPVB3FY68Xhn1TrkwV6o6MKH4pp+BEp1VzUXgrDsGQiZbyc36UCAwEAAaOBtTCBsjAOBgNVHQ8BAf8EBAMCBaAwCQYDVR0TBAIwADAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwNgYDVR0RBC8wLYIrdnN0cy12c3RzZ2hydC1naC12c28tb2F1dGgudmlzdWFsc3R1ZGlvLmNvbTAfBgNVHSMEGDAWgBRZBaZCR9ghvStfcWaGwuHGjrfTgzAdBgNVHQ4EFgQUWQWmQkfYIb0rX3FmhsLhxo6304MwDQYJKoZIhvcNAQELBQADggEBAGNdfALe6mdxQ67QL8GlW4dfFwvCX87JOeZThZ9uCj1+x1xUnywoR4o5q2DVI/JCvBRPn0BUb3dEVWLECXDHGjblesWZGMdSGYhMzWRQjVNmCYBC1ZM5QvonWCBcGkd72mZx0eFHnJCAP/TqEEpRvMHR+OOtSiZWV9zZpF1tf06AjKwT64F9V8PCmSIqPJXcTQXKKfkHZmGUk9AYF875+/FfzF89tCnT53UEh5BldFz0SAls+NhexbW/oOokBNCVqe+T2xXizktbFnFAFaomvwjVSvIeu3i/0Ygywl+3s5izMEsZ1T1ydIytv4FZf2JCHgRpmGPWJ5A7TpxuHSiE8Do="],"x5t":"eBZ_cn3sXYAd0ch4THBKHIgOwOE"},{"n":"wgCsNL8S6evSH_AHBsps2ccIHSwLpuEUGS9GYenGmGkSKyWefKsZheKl_84voiUgduuKcKA2aWQezp9338LjtlBmTHjopzAeU-Q3_IvqNf7BfrEAzEyp-ymdhNzPTE7Snmr5o_9AeiP1ZDBo35FaULgVUECJ3AzAM36zkURax3VNZRRZx1gb8lPUs9M5Yw6aZpHSOd6q_QzE8CP1OhGrAdoBzZ6ZCElon0kI-IuRLCwKptS7Yroi5-RtEKD2W458axNAQ36Yw93N8kInUC1QZDPrKd4QfYiG68ywjBoxp_bjNg5kh4LJmq1mwyGdNQV6F1Ew_jYlmou2Y8wvHQRJPQ","kty":"RSA","kid":"52F197C481DE70112C441B4A9B37B53C7FCF0DB5","alg":"RS256","e":"AQAB","use":"sig","x5c":["MIIDrDCCApSgAwIBAgIQLQnoXJ3HT6uPYvEofvOZ6zANBgkqhkiG9w0BAQsFADA2MTQwMgYDVQQDEyt2c3RzLXZzdHNnaHJ0LWdoLXZzby1vYXV0aC52aXN1YWxzdHVkaW8uY29tMB4XDTIxMTIwNjE5MDUyMloXDTIzMTIwNjE5MTUyMlowNjE0MDIGA1UEAxMrdnN0cy12c3RzZ2hydC1naC12c28tb2F1dGgudmlzdWFsc3R1ZGlvLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMIArDS/Eunr0h/wBwbKbNnHCB0sC6bhFBkvRmHpxphpEislnnyrGYXipf/OL6IlIHbrinCgNmlkHs6fd9/C47ZQZkx46KcwHlPkN/yL6jX+wX6xAMxMqfspnYTcz0xO0p5q+aP/QHoj9WQwaN+RWlC4FVBAidwMwDN+s5FEWsd1TWUUWcdYG/JT1LPTOWMOmmaR0jneqv0MxPAj9ToRqwHaAc2emQhJaJ9JCPiLkSwsCqbUu2K6IufkbRCg9luOfGsTQEN+mMPdzfJCJ1AtUGQz6yneEH2IhuvMsIwaMaf24zYOZIeCyZqtZsMhnTUFehdRMP42JZqLtmPMLx0EST0CAwEAAaOBtTCBsjAOBgNVHQ8BAf8EBAMCBaAwCQYDVR0TBAIwADAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwNgYDVR0RBC8wLYIrdnN0cy12c3RzZ2hydC1naC12c28tb2F1dGgudmlzdWFsc3R1ZGlvLmNvbTAfBgNVHSMEGDAWgBTTNQQWmG4PZZsdfMeamCH1YcyDZTAdBgNVHQ4EFgQU0zUEFphuD2WbHXzHmpgh9WHMg2UwDQYJKoZIhvcNAQELBQADggEBAK/d+HzBSRac7p6CTEolRXcBrBmmeJUDbBy20/XA6/lmKq73dgc/za5VA6Kpfd6EFmG119tl2rVGBMkQwRx8Ksr62JxmCw3DaEhE8ZjRARhzgSiljqXHlk8TbNnKswHxWmi4MD2/8QhHJwFj3X35RrdMM4R0dN/ojLlWsY9jXMOAvcSBQPBqttn/BjNzvn93GDrVafyX9CPl8wH40MuWS/gZtXeYIQg5geQkHCyP96M5Sy8ZABOo9MSIfPRw1F7dqzVuvliul9ZZGV2LsxmZCBtbsCkBau0amerigZjud8e9SNp0gaJ6wGhLbstCZIdaAzS5mSHVDceQzLrX2oe1h4k="],"x5t":"UvGXxIHecBEsRBtKmze1PH_PDbU"}]}`))

	queryStringParams := make(map[string]string)
	queryStringParams["owner"] = "varunsh-coder"
	queryStringParams["repo"] = "actions-playground"
	queryStringParams["runid"] = "2800694956"
	mockDynamoDbSvc := &mockDynamoDBClient{}

	tests := []struct {
		name    string
		args    args
		want    *GitHubWorkflowSecrets
		wantErr bool
	}{
		{name: "call from user", args: args{queryStringParams: queryStringParams, authHeader: "", svc: mockDynamoDbSvc},
			want: &GitHubWorkflowSecrets{Repo: "varunsh-coder/actions-playground", RunId: "2800694956", AreSecretsSet: true, Secrets: []Secret{{Name: "test"}}}, wantErr: false},
		{name: "call from action", args: args{queryStringParams: queryStringParams, authHeader: "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6ImVCWl9jbjNzWFlBZDBjaDRUSEJLSElnT3dPRSIsImtpZCI6Ijc4MTY3RjcyN0RFQzVEODAxREQxQzg3ODRDNzA0QTFDODgwRUMwRTEifQ.eyJqdGkiOiIzNGI0YTM1Ny05MjQ1LTRmYjgtOTlmYy00NDc3OWRiM2NmMzkiLCJzdWIiOiJyZXBvOnZhcnVuc2gtY29kZXIvYWN0aW9ucy1wbGF5Z3JvdW5kOnJlZjpyZWZzL2hlYWRzL21haW4iLCJhdWQiOiJodHRwczovL2dpdGh1Yi5jb20vdmFydW5zaC1jb2RlciIsInJlZiI6InJlZnMvaGVhZHMvbWFpbiIsInNoYSI6IjUzNmNmN2IwNGVlZGEyYmQ2ZGFmOTBjNTgzM2Q5ZTkwYjk1MTkyNGUiLCJyZXBvc2l0b3J5IjoidmFydW5zaC1jb2Rlci9hY3Rpb25zLXBsYXlncm91bmQiLCJyZXBvc2l0b3J5X293bmVyIjoidmFydW5zaC1jb2RlciIsInJlcG9zaXRvcnlfb3duZXJfaWQiOiIyNTAxNTkxNyIsInJ1bl9pZCI6IjI4MDA2OTQ5NTYiLCJydW5fbnVtYmVyIjoiNiIsInJ1bl9hdHRlbXB0IjoiMSIsInJlcG9zaXRvcnlfdmlzaWJpbGl0eSI6InB1YmxpYyIsInJlcG9zaXRvcnlfaWQiOiI0MzM5MDM3OTIiLCJhY3Rvcl9pZCI6IjI1MDE1OTE3IiwiYWN0b3IiOiJ2YXJ1bnNoLWNvZGVyIiwid29ya2Zsb3ciOiJQdWJsaXNoIFBhY2thZ2UgdG8gbnBtanMiLCJoZWFkX3JlZiI6IiIsImJhc2VfcmVmIjoiIiwiZXZlbnRfbmFtZSI6IndvcmtmbG93X2Rpc3BhdGNoIiwicmVmX3R5cGUiOiJicmFuY2giLCJqb2Jfd29ya2Zsb3dfcmVmIjoidmFydW5zaC1jb2Rlci9hY3Rpb25zLXBsYXlncm91bmQvLmdpdGh1Yi93b3JrZmxvd3MvbWZhX3JlbGVhc2UueW1sQHJlZnMvaGVhZHMvbWFpbiIsImlzcyI6Imh0dHBzOi8vdG9rZW4uYWN0aW9ucy5naXRodWJ1c2VyY29udGVudC5jb20iLCJuYmYiOjE2NTk2NjIzOTMsImV4cCI6MTY1OTY2MzI5MywiaWF0IjoxNjU5NjYyOTkzfQ.O-SRv44w8cHSsvQ40ntM5yqXTx4xLnp3koHZVwNcnes2DPGzbcXbf_qzmJqwpSVBqBjQUDS-nKLD_NgM8XSSgIQiTTIL0CBgZCb2FAwkYaVFWoMR38F1Z2OvHKz_WgsvaTX9thfMHyTe3gbFr1B8JSv2MeBQbFODCw7F1mkIPGPCd5wVAKjY3ECZp2JCmQ8nNvMtZj-HvuK5g3bXRpZASePufjhN2MP2y_ewGydWyNYIT6_sNIw8pab4eeD7VEaCTaxq4_yQkayPr49_xB5-g8H6LvY_aLMczJq9NpQMboEfFtlnQVQ90g4F7bFQd_cdMZPquKT0AJmDEsu04F1Hag", svc: mockDynamoDbSvc},
			want: &GitHubWorkflowSecrets{Repo: "varunsh-coder/actions-playground", RunId: "2800694956", AreSecretsSet: true, Secrets: []Secret{{Name: "test", Value: "123"}}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSecrets(tt.args.queryStringParams, tt.args.authHeader, tt.args.svc, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSecrets() = %v, want %v", got, tt.want)
			}
		})
	}
}
