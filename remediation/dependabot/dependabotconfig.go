package dependabot

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	dependabot "github.com/paulvollmer/dependabot-config-go"
	"gopkg.in/yaml.v3"
)

type UpdateDependabotConfigResponse struct {
	OriginalInput        string
	FinalOutput          string
	IsChanged            bool
	ConfigfileFetchError bool
}

type Ecosystem struct {
	PackageEcosystem string
	Directory        string
	Interval         string
}

type UpdateDependabotConfigRequest struct {
	Ecosystems []Ecosystem
	Content    string
}

// getIndentation returns the indentation level of the first list found in a given YAML string.
// If the YAML string is empty or invalid, or if no list is found, it returns an error.
func getIndentation(dependabotConfig string) (int, error) {
	// Initialize an empty YAML node
	t := yaml.Node{}

	// Unmarshal the YAML string into the node
	err := yaml.Unmarshal([]byte(dependabotConfig), &t)
	if err != nil {
		return 0, fmt.Errorf("unable to parse yaml: %w", err)
	}

	// Retrieve the top node of the YAML document
	topNode := t.Content
	if len(topNode) == 0 {
		return 0, errors.New("file provided is empty or invalid")
	}

	// Check for the first list and its indentation level
	for _, n := range topNode[0].Content {
		if n.Value == "" && n.Tag == "!!seq" {
			// Return the column of the first list found
			return n.Column, nil
		}
	}

	// Return an error if no list was found
	return 0, errors.New("no list found in yaml")
}

// UpdateDependabotConfig is used to update dependabot configuration and returns an UpdateDependabotConfigResponse.
func UpdateDependabotConfig(dependabotConfig string) (*UpdateDependabotConfigResponse, error) {
	var updateDependabotConfigRequest UpdateDependabotConfigRequest

	// Handle error in json unmarshalling
	err := json.Unmarshal([]byte(dependabotConfig), &updateDependabotConfigRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON from dependabotConfig: %v", err)
	}

	inputConfigFile := []byte(updateDependabotConfigRequest.Content)
	configMetadata := dependabot.New()
	err = configMetadata.Unmarshal(inputConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dependabot config: %v", err)
	}

	indentation := 3

	response := new(UpdateDependabotConfigResponse)
	response.FinalOutput = updateDependabotConfigRequest.Content
	response.OriginalInput = updateDependabotConfigRequest.Content
	response.IsChanged = false

	// Using strings.Builder for efficient string concatenation
	var finalOutput strings.Builder
	finalOutput.WriteString(response.FinalOutput)

	if updateDependabotConfigRequest.Content == "" {
		if len(updateDependabotConfigRequest.Ecosystems) == 0 {
			return response, nil
		}
		finalOutput.WriteString("version: 2\nupdates:")
	} else {
		if !strings.HasSuffix(response.FinalOutput, "\n") {
			finalOutput.WriteString("\n")
		}
		indentation, err = getIndentation(string(inputConfigFile))
		if err != nil {
			return nil, fmt.Errorf("failed to get indentation: %v", err)
		}
	}

	for _, Update := range updateDependabotConfigRequest.Ecosystems {
		updateAlreadyExist := false
		for _, update := range configMetadata.Updates {
			if update.PackageEcosystem == Update.PackageEcosystem && update.Directory == Update.Directory {
				updateAlreadyExist = true
				break
			}
		}

		if !updateAlreadyExist {
			item := dependabot.Update{
				PackageEcosystem: Update.PackageEcosystem,
				Directory:        Update.Directory,
				Schedule:         dependabot.Schedule{Interval: Update.Interval},
			}
			items := []dependabot.Update{item}
			addedItem, err := yaml.Marshal(items)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal update items: %v", err)
			}

			data, err := addIndentation(string(addedItem), indentation)
			if err != nil {
				return nil, fmt.Errorf("failed to add indentation: %v", err)
			}
			finalOutput.WriteString(data)
			response.IsChanged = true
		}
	}

	// Set FinalOutput to the built string
	response.FinalOutput = finalOutput.String()

	return response, nil
}

// addIndentation adds a certain number of spaces to the start of each line in the input string.
// It returns a new string with the added indentation.
func addIndentation(data string, indentation int) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	var finalData strings.Builder

	// Create the indentation string
	spaces := strings.Repeat(" ", indentation-1)

	finalData.WriteString("\n")

	// Add indentation to each line
	for scanner.Scan() {
		finalData.WriteString(spaces)
		finalData.WriteString(scanner.Text())
		finalData.WriteString("\n")
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error during scanning: %w", err)
	}

	return finalData.String(), nil
}
