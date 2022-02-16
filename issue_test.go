package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestCreateIssue(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/secure-workflows/issues?labels=knowledge-base&per_page=100&state=open",
		httpmock.NewStringResponder(200, `[
			{
			  "url": "https://api.github.com/repos/step-security/secure-workflows/issues/84",
			  "repository_url": "https://api.github.com/repos/step-security/secure-workflows",
			  "labels_url": "https://api.github.com/repos/step-security/secure-workflows/issues/84/labels{/name}",
			  "comments_url": "https://api.github.com/repos/step-security/secure-workflows/issues/84/comments",
			  "events_url": "https://api.github.com/repos/step-security/secure-workflows/issues/84/events",
			  "html_url": "https://github.com/step-security/secure-workflows/issues/84",
			  "id": 1074627768,
			  "node_id": "I_kwDOGM3SS85ADYS4",
			  "number": 84,
			  "title": "Add KB for actions/checkout1",
			  "state": "open",			  
			  "body": "e.g. ghcr.io\r\n"			  
			}
		  ]`))

	httpmock.RegisterResponder("POST", "https://api.github.com/repos/step-security/secure-workflows/issues",
		httpmock.NewStringResponder(200, `
			{
			  "url": "https://api.github.com/repos/step-security/secure-workflows/issues/84",
			  "repository_url": "https://api.github.com/repos/step-security/secure-workflows",
			  "labels_url": "https://api.github.com/repos/step-security/secure-workflows/issues/84/labels{/name}",
			  "comments_url": "https://api.github.com/repos/step-security/secure-workflows/issues/84/comments",
			  "events_url": "https://api.github.com/repos/step-security/secure-workflows/issues/84/events",
			  "html_url": "https://github.com/step-security/secure-workflows/issues/84",
			  "id": 1074627768,
			  "node_id": "I_kwDOGM3SS85ADYS4",
			  "number": 85,
			  "title": "Add KB for actions/checkout2",
			  "state": "open",			  
			  "body": "e.g. ghcr.io\r\n"			  
			}
		  `))

	tests := []struct {
		name    string
		Action  string
		PAT     string
		want    int
		wantErr bool
	}{
		{name: "not an action", Action: "", wantErr: true},
		{name: "action already in kb", Action: "actions/checkout", wantErr: true},
		{name: "PAT not set", Action: "actions/checkout1", wantErr: true},
		{name: "issue already exists", Action: "actions/checkout1", PAT: "123", want: 84},
		{name: "issue created", Action: "actions/checkout2", PAT: "123", want: 85},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.PAT != "" {
				os.Setenv("PAT", tt.PAT)
			} else {
				os.Setenv("PAT", "")
			}
			got, err := CreateIssue(tt.Action)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}
