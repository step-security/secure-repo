package workflow

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/jarcoal/httpmock"
)

func Test_getGitHubWorkflowContents(t *testing.T) {
	type args struct {
		queryStringParams map[string]string
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.github.com/repos/caolan/async/contents/.github/workflows/ci.yml?ref=master",
		httpmock.NewStringResponder(200, `{
			"name": "ci.yml",
			"path": ".github/workflows/ci.yml",
			"sha": "2b32ea48db98f4a607d23632a083bc031ac9f865",
			"size": 1926,
			"url": "https://api.github.com/repos/caolan/async/contents/.github/workflows/ci.yml?ref=master",
			"html_url": "https://github.com/caolan/async/blob/master/.github/workflows/ci.yml",
			"git_url": "https://api.github.com/repos/caolan/async/git/blobs/2b32ea48db98f4a607d23632a083bc031ac9f865",
			"download_url": "https://raw.githubusercontent.com/caolan/async/master/.github/workflows/ci.yml",
			"type": "file",
			"content": "bmFtZTogQ0kKCm9uOgogIHB1c2g6CiAgICBicmFuY2hlcy1pZ25vcmU6CiAg\nICAgIC0gImRlcGVuZGFib3QvKioiCiAgcHVsbF9yZXF1ZXN0OgoKam9iczoK\nICBsaW50OgogICAgcnVucy1vbjogdWJ1bnR1LWxhdGVzdAogICAgc3RlcHM6\nCiAgICAgIC0gbmFtZTog4qyH77iPIENoZWNrb3V0CiAgICAgICAgdXNlczog\nYWN0aW9ucy9jaGVja291dEB2MgoKICAgICAgLSBuYW1lOiDijpQgU2V0dXAg\nbm9kZSAke3sgbWF0cml4Lm5vZGUgfX0KICAgICAgICB1c2VzOiBhY3Rpb25z\nL3NldHVwLW5vZGVAdjIKICAgICAgICB3aXRoOgogICAgICAgICAgY2FjaGU6\nIG5wbQoKICAgICAgLSBuYW1lOiDwn5OlIERvd25sb2FkIGRlcHMKICAgICAg\nICBydW46IG5wbSBjaQoKICAgICAgLSBuYW1lOiDwn6eqIFJ1biBsaW50CiAg\nICAgICAgcnVuOiBucG0gcnVuIGxpbnQKCiAgYnVpbGQ6CiAgICBydW5zLW9u\nOiAke3sgbWF0cml4Lm9zIH19CiAgICBuZWVkczogbGludAogICAgc3RyYXRl\nZ3k6CiAgICAgIGZhaWwtZmFzdDogZmFsc2UKICAgICAgbWF0cml4OgogICAg\nICAgIG5vZGU6CiAgICAgICAgICAtIDEyCiAgICAgICAgICAtIDE0CiAgICAg\nICAgICAtIDE2CiAgICAgICAgICAtIDE3CiAgICAgICAgb3M6IFt1YnVudHUt\nbGF0ZXN0XQogICAgICAgIGJyb3dzZXI6CiAgICAgICAgICAtIEZpcmVmb3hI\nZWFkbGVzcwogICAgICAgIGluY2x1ZGU6CiAgICAgICAgICAtIG9zOiBtYWNv\ncy1sYXRlc3QKICAgICAgICAgICAgbm9kZTogMTYKICAgICAgICAgICAgYnJv\nd3NlcjogRmlyZWZveEhlYWRsZXNzCiAgICAgICAgICAtIG9zOiB3aW5kb3dz\nLWxhdGVzdAogICAgICAgICAgICBub2RlOiAxNgogICAgICAgICAgICBicm93\nc2VyOiBGaXJlZm94SGVhZGxlc3MKCiAgICBzdGVwczoKICAgICAgLSBuYW1l\nOiDwn5uRIENhbmNlbCBQcmV2aW91cyBSdW5zCiAgICAgICAgdXNlczogc3R5\nZmxlL2NhbmNlbC13b3JrZmxvdy1hY3Rpb25AMC45LjEKICAgICAgICB3aXRo\nOgogICAgICAgICAgYWNjZXNzX3Rva2VuOiAke3sgc2VjcmV0cy5HSVRIVUJf\nVE9LRU4gfX0KCiAgICAgIC0gbmFtZTog4qyH77iPIENoZWNrb3V0CiAgICAg\nICAgdXNlczogYWN0aW9ucy9jaGVja291dEB2MgoKICAgICAgLSBuYW1lOiDi\njpQgU2V0dXAgbm9kZSAke3sgbWF0cml4Lm5vZGUgfX0KICAgICAgICB1c2Vz\nOiBhY3Rpb25zL3NldHVwLW5vZGVAdjIKICAgICAgICB3aXRoOgogICAgICAg\nICAgbm9kZS12ZXJzaW9uOiAke3sgbWF0cml4Lm5vZGUgfX0KICAgICAgICAg\nIGNhY2hlOiBucG0KCiAgICAgIC0gbmFtZTog8J+TpSBEb3dubG9hZCBkZXBz\nCiAgICAgICAgcnVuOiBucG0gY2kKCiAgICAgIC0gbmFtZTogUnVuIGNvdmVy\nYWdlCiAgICAgICAgcnVuOiBucG0gdGVzdAoKICAgICAgLSBuYW1lOiBSdW4g\nYnJvd3NlciB0ZXN0cwogICAgICAgIGlmOiBtYXRyaXgubm9kZSA9PSAxNgog\nICAgICAgIHJ1bjogbnBtIHJ1biBtb2NoYS1icm93c2VyLXRlc3QgLS0gLS1i\ncm93c2VycyAke3sgbWF0cml4LmJyb3dzZXIgfX0gIC0tdGltZW91dCAxMDAw\nMAogICAgICAgIGVudjoKICAgICAgICAgIERJU1BMQVk6IDo5OS4wCgogICAg\nICAtIG5hbWU6IENvdmVyYWdlCiAgICAgICAgaWY6IG1hdHJpeC5vcyA9PSAn\ndWJ1bnR1LWxhdGVzdCcgJiYgbWF0cml4Lm5vZGUgPT0gJzE2JwogICAgICAg\nIHJ1bjogbnBtIHJ1biBjb3ZlcmFnZSAmJiBucHggbnljIHJlcG9ydCAtLXJl\ncG9ydGVyPWxjb3YKCiAgICAgIC0gbmFtZTogQ292ZXJhbGxzCiAgICAgICAg\naWY6IG1hdHJpeC5vcyA9PSAndWJ1bnR1LWxhdGVzdCcgJiYgbWF0cml4Lm5v\nZGUgPT0gJzE2JwogICAgICAgIHVzZXM6IGNvdmVyYWxsc2FwcC9naXRodWIt\nYWN0aW9uQDEuMS4zCiAgICAgICAgd2l0aDoKICAgICAgICAgICAgZ2l0aHVi\nLXRva2VuOiAke3sgc2VjcmV0cy5HSVRIVUJfVE9LRU4gfX0K\n",
			"encoding": "base64",
			"_links": {
			  "self": "https://api.github.com/repos/caolan/async/contents/.github/workflows/ci.yml?ref=master",
			  "git": "https://api.github.com/repos/caolan/async/git/blobs/2b32ea48db98f4a607d23632a083bc031ac9f865",
			  "html": "https://github.com/caolan/async/blob/master/.github/workflows/ci.yml"
			}
		  }`))

	expectedOutput, err := ioutil.ReadFile("../../testfiles/workflow-expected.yml")
	if err != nil {
		log.Fatal(err)
	}
	queryStringParams := make(map[string]string)
	queryStringParams["owner"] = "caolan"
	queryStringParams["repo"] = "async"
	queryStringParams["path"] = ".github/workflows/ci.yml"
	queryStringParams["branch"] = "master"

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "Public workflow", args: args{queryStringParams: queryStringParams}, want: string(expectedOutput), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGitHubWorkflowContents(tt.args.queryStringParams)
			if (err != nil) != tt.wantErr {
				t.Errorf("getGitHubWorkflowContents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getGitHubWorkflowContents() = %v, want %v", got, tt.want)
			}
		})
	}
}
