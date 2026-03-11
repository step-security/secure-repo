package dependabot

import (
	"bufio"
	"bytes"
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
	CoolDown         *dbCoolDown
}

type UpdateDependabotConfigRequest struct {
	Ecosystems  []Ecosystem
	Content     string
	Subtractive bool
}

// dbCoolDown represents the cooldown block, which the upstream dependabot package does not support.
type dbCoolDown struct {
	DefaultDays     int      `yaml:"default-days,omitempty"`
	SemverMajorDays int      `yaml:"semver-major-days,omitempty"`
	SemverMinorDays int      `yaml:"semver-minor-days,omitempty"`
	SemverPatchDays int      `yaml:"semver-patch-days,omitempty"`
	Include         []string `yaml:"include,omitempty"`
	Exclude         []string `yaml:"exclude,omitempty"`
}

// dbUpdate embeds the upstream dependabot.Update inline so all its fields are preserved,
// and extends it with the cooldown block.
type dbUpdate struct {
	dependabot.Update `yaml:",inline"`
	CoolDown          *dbCoolDown `yaml:"cooldown,omitempty"`
}

// dbConfig is the top-level dependabot config file structure backed by dbUpdate.
type dbConfig struct {
	Version int        `yaml:"version"`
	Updates []dbUpdate `yaml:"updates"`
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
	var cfg dbConfig
	err = yaml.Unmarshal(inputConfigFile, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal dependabot config: %v", err)
	}

	indentation := 3

	response := new(UpdateDependabotConfigResponse)
	response.FinalOutput = updateDependabotConfigRequest.Content
	response.OriginalInput = updateDependabotConfigRequest.Content
	response.IsChanged = false

	// In subtractive mode, update only the specified fields of existing entries.
	if updateDependabotConfigRequest.Subtractive {
		if updateDependabotConfigRequest.Content == "" {
			return response, nil
		}
		newContent, changed, err := updateSubtractiveFields(response.FinalOutput, updateDependabotConfigRequest.Ecosystems)
		if err != nil {
			return nil, fmt.Errorf("failed to apply subtractive update: %v", err)
		}
		response.FinalOutput = newContent
		response.IsChanged = changed
		return response, nil
	}

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
		for _, update := range cfg.Updates {
			if update.PackageEcosystem == Update.PackageEcosystem && (update.Directory == Update.Directory || update.Directory == Update.Directory+"/") {
				updateAlreadyExist = true
				break
			}
		}

		if !updateAlreadyExist {
			item := dbUpdate{
				Update: dependabot.Update{
					PackageEcosystem: Update.PackageEcosystem,
					Directory:        Update.Directory,
					Schedule:         dependabot.Schedule{Interval: Update.Interval},
				},
			}
			items := []dbUpdate{item}
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

// updateSubtractiveFields finds each ecosystem entry in the existing YAML config by
// PackageEcosystem + Directory, then updates only the non-empty fields from the request,
// leaving every other field of that entry unchanged.
func updateSubtractiveFields(content string, ecosystems []Ecosystem) (string, bool, error) {
	var cfg dbConfig
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return "", false, fmt.Errorf("failed to parse yaml: %w", err)
	}

	isChanged := false
	for _, eco := range ecosystems {
		for i, update := range cfg.Updates {
			if update.PackageEcosystem != eco.PackageEcosystem {
				continue
			}
			if update.Directory != eco.Directory && update.Directory != eco.Directory+"/" {
				continue
			}

			// Found the matching entry — update only non-empty fields.
			if eco.Interval != "" && cfg.Updates[i].Schedule.Interval != eco.Interval {
				cfg.Updates[i].Schedule.Interval = eco.Interval
				isChanged = true
			}

			if eco.CoolDown != nil {
				if cfg.Updates[i].CoolDown == nil {
					cfg.Updates[i].CoolDown = &dbCoolDown{}
				}
				existing := cfg.Updates[i].CoolDown
				cd := eco.CoolDown
				if cd.DefaultDays != 0 && existing.DefaultDays != cd.DefaultDays {
					existing.DefaultDays = cd.DefaultDays
					isChanged = true
				}
				if cd.SemverMajorDays != 0 && existing.SemverMajorDays != cd.SemverMajorDays {
					existing.SemverMajorDays = cd.SemverMajorDays
					isChanged = true
				}
				if cd.SemverMinorDays != 0 && existing.SemverMinorDays != cd.SemverMinorDays {
					existing.SemverMinorDays = cd.SemverMinorDays
					isChanged = true
				}
				if cd.SemverPatchDays != 0 && existing.SemverPatchDays != cd.SemverPatchDays {
					existing.SemverPatchDays = cd.SemverPatchDays
					isChanged = true
				}
				if len(cd.Include) > 0 {
					existing.Include = cd.Include
					isChanged = true
				}
				if len(cd.Exclude) > 0 {
					existing.Exclude = cd.Exclude
					isChanged = true
				}
			}
			break
		}
	}

	if !isChanged {
		return content, false, nil
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&cfg); err != nil {
		return "", false, fmt.Errorf("failed to marshal yaml: %w", err)
	}
	return buf.String(), true, nil
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
