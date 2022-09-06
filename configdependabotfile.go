package main

import (
	"bufio"
	"strings"

	dependabot "github.com/paulvollmer/dependabot-config-go"
	"gopkg.in/yaml.v2"
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
		comment := "Workflow files stored in the \ndefault location of `.github/workflows`"

		data = addIndentAndComment(data, comment)
		if err != nil {
			return nil, err
		}

		response.FinalOutput = response.FinalOutput + "\n" + data
		response.IsChanged = true
	}

	return response, nil
}

func addIndentAndComment(data string, comment string) string {
	scanner := bufio.NewScanner(strings.NewReader(data))
	cnt := 0
	finalData := "\n"
	for scanner.Scan() {
		if cnt == 1 {
			scanner2 := bufio.NewScanner(strings.NewReader(comment))
			for scanner2.Scan() {
				finalData += "    # " + scanner2.Text() + "\n"
			}
		}
		finalData += "  " + scanner.Text() + "\n"
		cnt++
	}
	return finalData
}
