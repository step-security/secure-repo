package main

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func pinDocker(action, jobName, inputYaml string) string {
	leftOfAt := strings.Split(action, "@")
	tag := leftOfAt[1]
	image := leftOfAt[0][9:]

	ref, err := name.ParseReference(image)
	if err != nil {
		return inputYaml
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return inputYaml
	}

	// Getting image digest
	imghash, err := img.Digest()
	if err != nil {
		return inputYaml
	}

	pinnedAction := fmt.Sprintf("%s@%s # %s", leftOfAt[0], imghash, tag)
	inputYaml = strings.ReplaceAll(inputYaml, action, pinnedAction)
	return inputYaml
}
