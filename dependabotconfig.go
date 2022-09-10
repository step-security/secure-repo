package main

import (
	"bufio"
	"strings"

	dependabot "github.com/paulvollmer/dependabot-config-go"
	"gopkg.in/yaml.v2"
)

type UpdateDependabotConfigResponse struct {
	OriginalInput        string
	FinalOutput          string
	IsChanged            bool
	ConfigfileFetchError bool
}

func UpdateDependabotConfig(dependabotConfig string) (*UpdateDependabotConfigResponse, error) {
	inputConfigFile := []byte(dependabotConfig)
	configMetadata := dependabot.New()
	err := configMetadata.Unmarshal(inputConfigFile)
	if err != nil {
		return nil, err
	}

	response := new(UpdateDependabotConfigResponse)
	response.FinalOutput = dependabotConfig
	response.OriginalInput = dependabotConfig
	response.IsChanged = false

	if !configMetadata.HasPackageEcosystem("github-actions") {
		item := dependabot.Update{}
		item.PackageEcosystem = "github-actions"
		item.Directory = "/"

		schedule := dependabot.Schedule{}
		schedule.Interval = "daily"

		item.Schedule = schedule
		items := []dependabot.Update{}
		items = append(items, item)
		addedItem, err := yaml.Marshal(items)
		data := string(addedItem)

		data = addIndentation(data)
		if err != nil {
			return nil, err
		}

		response.FinalOutput = response.FinalOutput + "\n" + data
		response.IsChanged = true
	}

	return response, nil
}

func addIndentation(data string) string {
	scanner := bufio.NewScanner(strings.NewReader(data))
	finalData := "\n"
	for scanner.Scan() {
		finalData += "  " + scanner.Text() + "\n"
	}
	return finalData
}
