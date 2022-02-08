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

func PinDockers(inputYaml string) (string, error) {
	workflow := Workflow{}

	err := yaml.Unmarshal([]byte(inputYaml), &workflow)
	if err != nil {
		return inputYaml, fmt.Errorf("unable to parse yaml %v", err)
	}

	out := inputYaml

	for jobName, job := range workflow.Jobs {

		for _, step := range job.Steps {
			if len(step.Uses) > 0 && strings.HasPrefix(step.Uses, "docker://") {
				out = pinDocker(step.Uses, jobName, out)
			}
		}
	}

	return out, nil
}

func pinDocker(action, jobName, inputYaml string) string {
	leftOfAt := strings.Split(action, ":")
	tag := leftOfAt[2]
	image := leftOfAt[1][2:]

	ref, err := name.ParseReference(image, name.WithDefaultTag(tag))
	if err != nil {
		return inputYaml
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithTransport(Tr))
	if err != nil {
		return inputYaml
	}

	// Getting image digest
	imghash, err := img.Digest()
	if err != nil {
		return inputYaml
	}

	pinnedAction := fmt.Sprintf("%s:%s@%s # %s", leftOfAt[0], leftOfAt[1], imghash.String(), tag)
	inputYaml = strings.ReplaceAll(inputYaml, action, pinnedAction)
	return inputYaml
}
