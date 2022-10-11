package main

import (
	"bufio"
	"encoding/json"
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

type Ecosystem struct {
	PackageEcosystem string
	Directory        string
	Interval         string
}

type UpdateDependabotConfigRequest struct {
	Ecosystems []Ecosystem
	Content    string
}

func UpdateDependabotConfig(dependabotConfig string) (*UpdateDependabotConfigResponse, error) {
	var updateDependabotConfigRequest UpdateDependabotConfigRequest
	json.Unmarshal([]byte(dependabotConfig), &updateDependabotConfigRequest)
	inputConfigFile := []byte(updateDependabotConfigRequest.Content)
	configMetadata := dependabot.New()
	err := configMetadata.Unmarshal(inputConfigFile)
	if err != nil {
		return nil, err
	}

	response := new(UpdateDependabotConfigResponse)
	response.FinalOutput = updateDependabotConfigRequest.Content
	response.OriginalInput = updateDependabotConfigRequest.Content
	response.IsChanged = false

	if updateDependabotConfigRequest.Content == "" {
		if len(updateDependabotConfigRequest.Ecosystems) == 0 {
			return response, nil
		}
		response.FinalOutput = "version: 2\nupdates:"
	} else {
		response.FinalOutput += "\n"
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
			item := dependabot.Update{}
			item.PackageEcosystem = Update.PackageEcosystem
			item.Directory = Update.Directory

			schedule := dependabot.Schedule{}
			schedule.Interval = Update.Interval

			item.Schedule = schedule
			items := []dependabot.Update{}
			items = append(items, item)
			addedItem, err := yaml.Marshal(items)
			data := string(addedItem)

			data = addIndentation(data)
			if err != nil {
				return nil, err
			}
			response.FinalOutput = response.FinalOutput + data
			response.IsChanged = true
		}
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
