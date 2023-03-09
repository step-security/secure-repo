package docker

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/asottile/dockerfile"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var Tr http.RoundTripper = remote.DefaultTransport

type SecureDockerfileResponse struct {
	OriginalInput        string
	FinalOutput          string
	IsChanged            bool
	DockerfileFetchError bool
}

func SecureDockerFile(inputDockerFile string) (*SecureDockerfileResponse, error) {
	reader := strings.NewReader(inputDockerFile)
	cmds, err := dockerfile.ParseReader(reader)
	if err != nil {
		return nil, err
	}

	response := new(SecureDockerfileResponse)
	response.FinalOutput = inputDockerFile
	response.OriginalInput = inputDockerFile
	response.IsChanged = false

	for _, c := range cmds {
		if strings.Contains(c.Cmd, "FROM") && strings.Contains(c.Value[0], ":") {
			// For being fixable
			// image must have either:
			// * 1 colon without sha256
			// * 2 colon with sha256
			temp := c.Value[0]
			var image string
			var tag string
			isPinned := false
			if strings.Contains(temp, ":") && !strings.Contains(temp, "sha256") {
				// case activates if image like: python:3.7
				split := strings.Split(temp, ":")
				image = split[0]
				tag = split[1]
			}
			if strings.Count(temp, ":") == 2 && strings.Contains(temp, "sha256") {

				// case activates if image like: python:3.7@sha256:v2
				t := strings.Split(temp, "@")
				if len(t[1]) != 71 {
					tt := strings.Split(t[0], ":")
					image = tt[0]
					tag = tt[1]
				} else {
					isPinned = true
				}
			}
			if strings.Count(temp, ":") == 1 && strings.Contains(temp, "sha256") {
				// is already pinned
				isPinned = true
			}
			if !isPinned {
				sha, err := getSHA(image, tag)
				if err != nil {
					return nil, err
				}
				new_cmd := strings.ReplaceAll(c.Original, c.Value[0], fmt.Sprintf("%s:%s@%s", image, tag, sha))
				response.FinalOutput = strings.ReplaceAll(response.FinalOutput, c.Original, new_cmd)
				// Revert the extra hash for already pinned docker images
				response.FinalOutput = strings.ReplaceAll(response.FinalOutput, new_cmd+"@", c.Original+"@")
				response.IsChanged = true

			}

		}
	}

	return response, nil
}
func getSHA(image string, tag string) (string, error) {

	ref, err := name.ParseReference(image, name.WithDefaultTag(tag))
	if err != nil {
		return "", err
	}
	desc, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithTransport(Tr))

	if err != nil {
		return "", err
	}
	return desc.Digest.String(), nil
}
