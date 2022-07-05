package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"gopkg.in/yaml.v3"
)

var Tr http.RoundTripper = remote.DefaultTransport

func PinDocker(inputYaml string) (string, bool, error) {
	updated := false
	workflow := Workflow{}

	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return inputYaml, updated, fmt.Errorf("unable to parse yaml %v", err)
	}

	out := inputYaml

	for jobName, job := range workflow.Jobs {

		for _, step := range job.Steps {
			if len(step.Uses) > 0 && strings.HasPrefix(step.Uses, "docker://") {
				localUpdated := false
				out, localUpdated = pinDocker(step.Uses, jobName, out)
				updated = updated || localUpdated
			}
		}
	}

	return out, updated, nil
}

func pinDocker(action, jobName, inputYaml string) (string, bool) {
	updated := false
	leftOfAt := strings.Split(action, ":")
	tag := leftOfAt[2]
	image := leftOfAt[1][2:]

	ref, err := name.ParseReference(image, name.WithDefaultTag(tag))
	if err != nil {
		return inputYaml, updated
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithTransport(Tr))
	if err != nil {
		//TODO: Log the error
		return inputYaml, updated
	}

	// Getting image digest
	imghash, err := img.Digest()
	if err != nil {
		return inputYaml, updated
	}

	pinnedAction := fmt.Sprintf("%s:%s@%s # %s", leftOfAt[0], leftOfAt[1], imghash.String(), tag)
	inputYaml = strings.ReplaceAll(inputYaml, action, pinnedAction)
	updated = !strings.EqualFold(action, pinnedAction)
	return inputYaml, updated
}
