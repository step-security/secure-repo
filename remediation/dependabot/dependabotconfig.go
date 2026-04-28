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
	PackageEcosystem              string
	Directory                     string
	Directories                   []string                          `json:",omitempty"`
	Interval                      string
	CoolDown                      *CoolDown                         `json:",omitempty"`
	Groups                        map[string]Group                  `json:",omitempty"`
	Day                           string                            `json:",omitempty"`
	Time                          string                            `json:",omitempty"`
	Timezone                      string                            `json:",omitempty"`
	Allow                         []dependabot.Allow                `json:",omitempty"`
	Assignees                     []string                          `json:",omitempty"`
	CommitMessage                 *dependabot.CommitMessage         `json:",omitempty"`
	Ignore                        []dependabot.Ignore               `json:",omitempty"`
	Labels                        []string                          `json:",omitempty"`
	Milestone                     *int                              `json:",omitempty"`
	OpenPullRequestsLimit         *int                              `json:",omitempty"`
	PullRequestBranchName         *dependabot.PullRequestBranchName `json:",omitempty"`
	RebaseStrategy                string                            `json:",omitempty"`
	Reviewers                     []string                          `json:",omitempty"`
	TargetBranch                  string                            `json:",omitempty"`
	VersioningStrategy            string                            `json:",omitempty"`
	Registries                    []string                          `json:",omitempty"`
	ExcludePaths                  []string                          `json:",omitempty"`
	Vendor                        *bool                             `json:",omitempty"`
	InsecureExternalCodeExecution string                            `json:",omitempty"`
	MultiEcosystemGroup           string                            `json:",omitempty"`
	EnableBetaEcosystems          *bool                             `json:",omitempty"`
}

type UpdateDependabotConfigRequest struct {
	Ecosystems  []Ecosystem
	Content     string
	Subtractive bool
}

// CoolDown represents the cooldown block, which the upstream dependabot package does not support.
type CoolDown struct {
	DefaultDays     int      `yaml:"default-days,omitempty"`
	SemverMajorDays int      `yaml:"semver-major-days,omitempty"`
	SemverMinorDays int      `yaml:"semver-minor-days,omitempty"`
	SemverPatchDays int      `yaml:"semver-patch-days,omitempty"`
	Include         []string `yaml:"include,omitempty"`
	Exclude         []string `yaml:"exclude,omitempty"`
}

// Group represents a single entry in the groups block.
type Group struct {
	AppliesTo       string   `yaml:"applies-to,omitempty"`
	Patterns        []string `yaml:"patterns,omitempty"`
	ExcludePatterns []string `yaml:"exclude-patterns,omitempty"`
	DependencyType  string   `yaml:"dependency-type,omitempty"`
	UpdateTypes     []string `yaml:"update-types,omitempty"`
	GroupBy         string   `yaml:"group-by,omitempty"`
}

// ExtendedUpdate embeds the upstream dependabot.Update inline so all its fields are preserved,
// and extends it with fields the upstream library does not support.
type ExtendedUpdate struct {
	dependabot.Update             `yaml:",inline"`
	Directories                   []string         `yaml:"directories,omitempty"`
	Groups                        map[string]Group `yaml:"groups,omitempty"`
	CoolDown                      *CoolDown        `yaml:"cooldown,omitempty"`
	Registries                    []string         `yaml:"registries,omitempty"`
	ExcludePaths                  []string         `yaml:"exclude-paths,omitempty"`
	Vendor                        *bool            `yaml:"vendor,omitempty"`
	InsecureExternalCodeExecution string           `yaml:"insecure-external-code-execution,omitempty"`
	MultiEcosystemGroup           string           `yaml:"multi-ecosystem-group,omitempty"`
	EnableBetaEcosystems          *bool            `yaml:"enable-beta-ecosystems,omitempty"`
}

// Config is the top-level dependabot config file structure backed by Update.
type Config struct {
	Version int              `yaml:"version"`
	Updates []ExtendedUpdate `yaml:"updates"`
}

// matchesEcosystem checks whether an existing config entry matches the requested
// ecosystem by package-ecosystem and directory (singular or plural).
func matchesEcosystem(update ExtendedUpdate, eco Ecosystem) bool {
	if update.PackageEcosystem != eco.PackageEcosystem {
		return false
	}
	// Match by singular directory.
	if update.Directory != "" && (update.Directory == eco.Directory || update.Directory == eco.Directory+"/") {
		return true
	}
	// Match by plural directories: if any of the existing directories matches eco.Directory.
	for _, d := range update.Directories {
		if d == eco.Directory || d == eco.Directory+"/" {
			return true
		}
	}
	return false
}

// ecosystemToExtendedUpdate converts an Ecosystem API input into an ExtendedUpdate
// suitable for YAML marshaling. Used by both additive and subtractive (toAdd) paths.
func ecosystemToExtendedUpdate(eco Ecosystem) ExtendedUpdate {
	item := ExtendedUpdate{
		Update: dependabot.Update{
			PackageEcosystem: eco.PackageEcosystem,
			Directory:        eco.Directory,
			Schedule:         dependabot.Schedule{Interval: eco.Interval, Day: eco.Day, Time: eco.Time, Timezone: eco.Timezone},
		},
		Directories: eco.Directories,
		Groups:      eco.Groups,
		CoolDown:    eco.CoolDown,
	}

	for _, a := range eco.Allow {
		item.Update.Allow = append(item.Update.Allow, &dependabot.Allow{
			DependencyName: a.DependencyName,
			DependencyType: a.DependencyType,
		})
	}
	for _, a := range eco.Assignees {
		s := a
		item.Update.Assignees = append(item.Update.Assignees, &s)
	}
	if eco.CommitMessage != nil {
		item.Update.CommitMessage = &dependabot.CommitMessage{
			Prefix:            eco.CommitMessage.Prefix,
			PrefixDevelopment: eco.CommitMessage.PrefixDevelopment,
			Include:           eco.CommitMessage.Include,
		}
	}
	for _, ig := range eco.Ignore {
		item.Update.Ignore = append(item.Update.Ignore, &dependabot.Ignore{
			DependencyName: ig.DependencyName,
			Versions:       ig.Versions,
		})
	}
	for _, l := range eco.Labels {
		s := l
		item.Update.Labels = append(item.Update.Labels, &s)
	}
	item.Update.Milestone = eco.Milestone
	item.Update.OpenPullRequestsLimit = eco.OpenPullRequestsLimit
	if eco.PullRequestBranchName != nil {
		item.Update.PullRequestBranchName = &dependabot.PullRequestBranchName{
			Separator: eco.PullRequestBranchName.Separator,
		}
	}
	if eco.RebaseStrategy != "" {
		s := eco.RebaseStrategy
		item.Update.RebaseStrategy = &s
	}
	for _, r := range eco.Reviewers {
		s := r
		item.Update.Reviewers = append(item.Update.Reviewers, &s)
	}
	if eco.TargetBranch != "" {
		s := eco.TargetBranch
		item.Update.TargetBranch = &s
	}
	if eco.VersioningStrategy != "" {
		s := eco.VersioningStrategy
		item.Update.VersioningStrategy = &s
	}

	// ExtendedUpdate-only fields (not in the upstream library's Update struct).
	item.Registries = eco.Registries
	item.ExcludePaths = eco.ExcludePaths
	item.Vendor = eco.Vendor
	if eco.InsecureExternalCodeExecution != "" {
		item.InsecureExternalCodeExecution = eco.InsecureExternalCodeExecution
	}
	if eco.MultiEcosystemGroup != "" {
		item.MultiEcosystemGroup = eco.MultiEcosystemGroup
	}
	item.EnableBetaEcosystems = eco.EnableBetaEcosystems

	return item
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
	var cfg Config
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
		subtractiveIndent, err := getIndentation(string(inputConfigFile))
		if err != nil {
			return nil, fmt.Errorf("failed to get indentation: %v", err)
		}
		newContent, changed, err := updateSubtractiveFields(response.FinalOutput, updateDependabotConfigRequest.Ecosystems, cfg, subtractiveIndent-1)
		if err != nil {
			return nil, fmt.Errorf("failed to apply subtractive update: %v", err)
		}
		response.FinalOutput = newContent
		response.IsChanged = changed
		return response, nil
	}

	if updateDependabotConfigRequest.Content == "" {
		// Empty content: build from scratch using string concatenation.
		if len(updateDependabotConfigRequest.Ecosystems) == 0 {
			return response, nil
		}
		var finalOutput strings.Builder
		finalOutput.WriteString("version: 2\nupdates:")
		for _, eco := range updateDependabotConfigRequest.Ecosystems {
			item := ecosystemToExtendedUpdate(eco)
			items := []ExtendedUpdate{item}
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
		response.FinalOutput = finalOutput.String()
	} else {
		// Non-empty content: insert new entries at the end of the updates section
		// so that sibling top-level keys like registries are preserved in place.
		indentation, err = getIndentation(string(inputConfigFile))
		if err != nil {
			return nil, fmt.Errorf("failed to get indentation: %v", err)
		}

		var rootNode yaml.Node
		if err := yaml.Unmarshal(inputConfigFile, &rootNode); err != nil {
			return nil, fmt.Errorf("failed to parse yaml for insertion point: %v", err)
		}
		if len(rootNode.Content) == 0 {
			return nil, fmt.Errorf("failed to parse yaml: document is empty")
		}
		docNode := rootNode.Content[0]
		updatesNode := findMappingValue(docNode, "updates")
		if updatesNode == nil || updatesNode.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("missing or invalid 'updates' section in dependabot config")
		}

		inputLines := strings.Split(response.FinalOutput, "\n")
		updatesLastLine := findLastLine(updatesNode)
		lineOffset := 0

		for _, eco := range updateDependabotConfigRequest.Ecosystems {
			updateAlreadyExist := false
			for _, update := range cfg.Updates {
				if matchesEcosystem(update, eco) {
					updateAlreadyExist = true
					break
				}
			}

			if !updateAlreadyExist {
				// If an existing entry uses directories (plural) for the same ecosystem,
				// append the new directory to that list instead of creating a new entry.
				appendedToDirectories := false
				for i, update := range cfg.Updates {
					if update.PackageEcosystem == eco.PackageEcosystem && len(update.Directories) > 0 {
						entryNode := updatesNode.Content[i]
						dirsNode := findMappingValue(entryNode, "directories")
						if dirsNode != nil {
							newDirs := append(update.Directories, eco.Directory)
							newLines, netChange, ch := replaceSequence(inputLines, dirsNode, newDirs, lineOffset)
							if ch {
								inputLines = newLines
								lineOffset += netChange
								response.IsChanged = true
							}
							appendedToDirectories = true
						}
						break
					}
				}

				if !appendedToDirectories {
					item := ecosystemToExtendedUpdate(eco)
					items := []ExtendedUpdate{item}
					addedItem, err := yaml.Marshal(items)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal update items: %v", err)
					}
					data, err := addIndentation(string(addedItem), indentation)
					if err != nil {
						return nil, fmt.Errorf("failed to add indentation: %v", err)
					}

					// Trim trailing newline to avoid double blank lines when content
					// follows after the updates section (e.g. registries block).
					dataLines := strings.Split(strings.TrimRight(data, "\n"), "\n")
					insertAt := updatesLastLine + lineOffset
					inputLines = insertAfterLine(inputLines, insertAt, dataLines)
					lineOffset += len(dataLines)
					response.IsChanged = true
				}
			}
		}

		response.FinalOutput = strings.Join(inputLines, "\n")
		if !strings.HasSuffix(response.FinalOutput, "\n") {
			response.FinalOutput += "\n"
		}
	}

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
