package main

import (
	dependabot "github.com/paulvollmer/dependabot-config-go"
)

type configDependabotResponse struct {
	OriginalInput        string
	FinalOutput          string
	IsChanged            bool
	configfileFetchError bool
}

func configDependabot(configDependabotFile string) (*configDependabotResponse, error) {
	inputConfigFile := []byte(configDependabotFile)
	configMetadata := dependabot.New()
	err := configMetadata.Unmarshal(inputConfigFile)
	if err != nil {
		return nil, err
	}

	response := new(configDependabotResponse)
	response.FinalOutput = configDependabotFile
	response.OriginalInput = configDependabotFile
	response.IsChanged = false
	response.configfileFetchError = false

	if !configMetadata.HasPackageEcosystem("github-actions") {
		item := dependabot.Update{}
		item.PackageEcosystem = "github-actions"
		item.Directory = "/"

		schedule := dependabot.Schedule{}
		schedule.Interval = "daily"

		item.Schedule = schedule
		configMetadata.AddUpdate(item)
		data, err := configMetadata.Marshal()
		if err != nil {
			return nil, err
		}
		response.FinalOutput = string(data)
		response.IsChanged = true
	}

	return response, nil
}
