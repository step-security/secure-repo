package pin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
)

var (
	githubImmutableActionArtifactType = "application/vnd.github.actions.package.v1+json"
	semanticTagRegex                  = regexp.MustCompile(`v[0-9]+\.[0-9]+\.[0-9]+$`)
)

type ociManifest struct {
	ArtifactType string `json:"artifactType"`
}

// isImmutableAction checks if the action is an immutable action or not
// It queries the OCI manifest for the action and checks if the artifact type is "application/vnd.github.actions.package.v1+json"
//
// Example usage:
//
//	# Immutable action (returns true)
//	isImmutableAction("actions/checkout@v4.2.2")
//
//	# Non-Immutable action (returns false)
//	isImmutableAction("actions/checkout@v4.2.3")
//
// REF - https://github.com/actions/publish-immutable-action/issues/216#issuecomment-2549914784
func IsImmutableAction(action string) bool {

	artifactType, err := getOCIImageArtifactTypeForGhAction(action)
	if err != nil {
		// log the error
		logrus.WithFields(logrus.Fields{"action": action}).WithError(err).Error("error in getting OCI manifest for image")
		return false
	}

	if artifactType == githubImmutableActionArtifactType {
		return true
	}
	return false

}

// getOCIImageArtifactTypeForGhAction retrieves the artifact type from a GitHub Action's OCI manifest.
// This function is used to determine if an action is immutable by checking its artifact type.
//
// Example usage:
//
//	# Immutable action (returns "application/vnd.github.actions.package.v1+json", nil)
//	artifactType, err := getOCIImageArtifactTypeForGhAction("actions/checkout@v4.2.2")
//
// Returns:
//   - artifactType: The artifact type string from the OCI manifest
//   - error: An error if the action format is invalid or if there's a problem retrieving the manifest
func getOCIImageArtifactTypeForGhAction(action string) (string, error) {

	// Split the action into parts (e.g., "actions/checkout@v2" -> ["actions/checkout", "v2"])
	parts := strings.Split(action, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid action format")
	}

	// For bundled actions like github/codeql-action/analyze@v3,
	// we only need the repository part (github/codeql-action) to check for immutability
	actionPath := parts[0]
	if strings.Count(parts[0], "/") > 1 {
		pathParts := strings.Split(parts[0], "/")
		actionPath = strings.Join(pathParts[:2], "/")
	}

	// convert v1.x.x to 1.x.x which is
	// use regexp to match tag version format and replace v in prefix
	// as immutable actions image tag is in format 1.x.x (without v prefix)
	// REF - https://github.com/actions/publish-immutable-action/issues/216#issuecomment-2549914784
	if semanticTagRegex.MatchString(parts[1]) {
		// v1.x.x -> 1.x.x
		parts[1] = strings.TrimPrefix(parts[1], "v")
	}

	// Convert GitHub action to GHCR image reference using proper OCI reference format
	image := fmt.Sprintf("ghcr.io/%s:%s", actionPath, parts[1])
	imageManifest, err := getOCIManifestForImage(image)
	if err != nil {
		return "", err
	}

	var ociManifest ociManifest
	err = json.Unmarshal([]byte(imageManifest), &ociManifest)
	if err != nil {
		return "", err
	}
	return ociManifest.ArtifactType, nil
}

// getOCIManifestForImage retrieves the artifact type from the OCI image manifest
func getOCIManifestForImage(imageRef string) (string, error) {

	// Parse the image reference
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", fmt.Errorf("error parsing reference: %v", err)
	}

	// Get the image manifest
	desc, err := remote.Get(ref, remote.WithTransport(http.DefaultTransport))
	if err != nil {
		return "", fmt.Errorf("error getting manifest: %v", err)
	}

	return string(desc.Manifest), nil
}
