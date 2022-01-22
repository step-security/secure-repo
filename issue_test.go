package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/google/go-github/v38/github"
	"github.com/jarcoal/httpmock"
)

func TestCreateIssue(t *testing.T) {
	type args struct {
		step  *GitHubJobStepOut
		owner string
		repo  string
		runid string
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/step-security/secure-workflows/issues?labels=knowledge-base&per_page=100&state=all",
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
		args    args
		PAT     string
		want    int
		wantErr bool
	}{
		{name: "not an action", args: args{step: &GitHubJobStepOut{Action: ""}}, wantErr: true},
		{name: "action already in kb", args: args{step: &GitHubJobStepOut{Action: "actions/checkout"}}, wantErr: true},
		{name: "PAT not set", args: args{step: &GitHubJobStepOut{Action: "actions/checkout1"}}, wantErr: true},
		{name: "issue already exists", args: args{step: &GitHubJobStepOut{Action: "actions/checkout1"}}, PAT: "123", want: 84},
		{name: "issue created", args: args{step: &GitHubJobStepOut{Action: "actions/checkout2"}}, PAT: "123", want: 85},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.PAT != "" {
				os.Setenv("PAT", tt.PAT)
			}
			got, err := CreateIssue(tt.args.step, tt.args.owner, tt.args.repo, tt.args.runid)
			if tt.PAT != "" {
				os.Setenv("PAT", "")
			}
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

func Test_createIssue(t *testing.T) {
	type args struct {
		step  *GitHubJobStepOut
		owner string
		repo  string
		runid string
	}
	tests := []struct {
		name    string
		args    args
		want    *github.Issue
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createIssue(tt.args.step, tt.args.owner, tt.args.repo, tt.args.runid)
			if (err != nil) != tt.wantErr {
				t.Errorf("createIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}
