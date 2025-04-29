package maintainedactions

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Release struct {
	TagName string `json:"tag_name"`
}

func GetLatestRelease(ownerRepo string) (string, error) {
	// Build the URL dynamically and add `/actions` at the end
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", ownerRepo)
	fmt.Println("url ", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error fetching release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("non-200 response: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("error parsing JSON: %w", err)
	}

	return release.TagName, nil
}
