package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
)

func Test_getRepo(t *testing.T) {
	type args struct {
		action string
	}
	tests := []struct {
		name string
		args args
		want Repo
	}{
		// TODO: Add test cases.
		{name: "simple action", args: args{action: "step-security/harden-runner"}, want: Repo{owner: "step-security", repo: "harden-runner"}},
		{name: "inner action", args: args{action: "step-security/harden-runner/inner-action"}, want: Repo{owner: "step-security", repo: "harden-runner/inner-action"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRepo(tt.args.action)
			t.Log(got)
			if !strings.EqualFold(tt.want.owner, got.owner) && !strings.EqualFold(tt.want.repo, got.repo) {
				t.Errorf("getRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreatePR(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	os.Setenv("PAT", "temp_pat")

	httpmock.RegisterResponder("POST", "https://api.github.com/repos/step-security/secure-repo/actions/workflows/kbanalysis.yml/dispatches", httpmock.NewStringResponder(204, ""))

	tests := []struct {
		name    string
		Action  string
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "not an action", Action: "", wantErr: true},
		{name: "action already in kb", Action: "actions/checkout", wantErr: true},
		{name: "action not present", Action: "step-security/no-such-action", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreatePR(tt.Action); (err != nil) != tt.wantErr {
				t.Errorf("CreatePR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
