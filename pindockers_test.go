package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestDockerActions(t *testing.T) {
	const inputDirectory = "./testfiles/pindockers/input"
	const outputDirectory = "./testfiles/pindockers/output"
	files, err := ioutil.ReadDir(inputDirectory)
	if err != nil {
		log.Fatal(err)
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	saveTr := Tr
	defer func() { Tr = saveTr }()
	Tr = httpmock.DefaultTransport

	// add Table-Driven Tests
	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/",
		httpmock.NewStringResponder(200, `{
		}`))

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/",
		httpmock.NewStringResponder(200, `{
		}`))

	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/step-security/integration-test/int/manifests/latest",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, `{
				"digest": "sha256:f1f95204dc1f12a41eaf41080185e2d289596b3e7637a8c50a3f6fbe17f99649",
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"size": 1568,
				"config": {
					"digest": "sha256:d1ea12d5b843717d90bca617e78609dbcc2ebcc9386c9edd98ecdb1427469b43",
					"mediaType": "application/vnd.docker.container.image.v1+json",
					"size": 2868
				}
			}`)
			resp.Header.Add("Docker-Content-Digest", "sha256:f1f95204dc1f12a41eaf41080185e2d289596b3e7637a8c50a3f6fbe17f99649")
			return resp, nil
		},
	)

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/gcp-runtimes/container-structure-test/manifests/latest",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, `{
				"digest": "sha256:4affda1c8f058f8d6c86dcad965cdb438a3d1d9a982828ff6737ea492b6bc8ce",
				"schemaVersion": 2,
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"config": {
					"mediaType": "application/vnd.docker.container.image.v1+json",
					"size": 2262,
					"digest": "sha256:5ba40d8c6e46b72e8f1ef20a8a4b7905546286a1036abe48f8854ca5b34aa41a"
				}
			}`)
			resp.Header.Add("Docker-Content-Digest", "sha256:4affda1c8f058f8d6c86dcad965cdb438a3d1d9a982828ff6737ea492b6bc8ce")
			return resp, nil
		},
	)

	for _, f := range files {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		output, err := PinDockers(string(input))

		if err != nil {
			t.Errorf("Error: %v", err)
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		if output != string(expectedOutput) {
			t.Errorf("test failed %s did not match expected output\n%s", f.Name(), output)
		}
	}
}
