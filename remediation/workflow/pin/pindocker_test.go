package pin

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestDockerActions(t *testing.T) {

	CI := os.Getenv("CI")
	if len(CI) == 0 {
		// Only run on GitHub Actions workflow, since local docker config might interfere with test
		log.Println("TestDockerActions: CI not set, skipping")
		return
	}

	const inputDirectory = "../../../testfiles/pindockers/input"
	const outputDirectory = "../../../testfiles/pindockers/output"
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
	//Ping Docker Image
	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/",
		httpmock.NewStringResponder(200, `{
		}`))

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/",
		httpmock.NewStringResponder(200, `{
		}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/",
		httpmock.NewStringResponder(200, `{
		}`))

	//Fetch menifest file
	//step.1> Get Token by this api call (https://auth.docker.io/token?service=<service>&scope=repository:<name>:<tag>)
	//step.2> Get manifest file by setting header as (Authorization : Bearer <token>) and using this api call (https://<service>/v2/<name>/manifests/<tag>)
	//step.3> Download response and save it in this path (testfiles/pindockers/response/<name>.txt)
	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/step-security/integration-test/int/manifests/latest",
		httpmock.NewStringResponder(200, httpmock.File("../../../testfiles/pindockers/response/ghcrResponse.json").String()))

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/gcp-runtimes/container-structure-test/manifests/latest",
		httpmock.NewStringResponder(200, httpmock.File("../../../testfiles/pindockers/response/gcrResponse.json").String()))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/markstreet/conker/manifests/latest",
		httpmock.NewStringResponder(200, httpmock.File("../../../testfiles/pindockers/response/dockerResponse.json").String()))

	for _, f := range files {
		input, err := ioutil.ReadFile(path.Join(inputDirectory, f.Name()))

		if err != nil {
			log.Fatal(err)
		}

		output, _, err := PinDocker(string(input))

		if err != nil {
			t.Errorf("Error not expected")
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
