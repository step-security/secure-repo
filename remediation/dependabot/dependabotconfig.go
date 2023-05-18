package dependabot

import (
	"bufio"
	"encoding/json"
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

func getIndentation(dependabotConfig string) (int, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(dependabotConfig), &t)
	if err != nil {
		return 0, fmt.Errorf("unable to parse yaml %v", err)
	}

	column := 0
	topNode := t.Content
	if len(topNode) == 0 {
		return 0, fmt.Errorf("file provided is Empty")
	}
	for _, n := range topNode[0].Content {
		if n.Value == "" && n.Tag == "!!seq" {
			column = n.Column
			break
		}
	}
	return column, nil
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

	indentation := 3

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
		indentation, err = getIndentation(string(inputConfigFile))
		if err != nil {
			return nil, err
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

			data = addIndentation(data, indentation)
			if err != nil {
				return nil, err
			}
			response.FinalOutput = response.FinalOutput + data
			response.IsChanged = true
		}
	}

	return response, nil
}

func addIndentation(data string, indentation int) string {
	scanner := bufio.NewScanner(strings.NewReader(data))
	finalData := "\n"
	spaces := ""
	for i := 0; i < indentation-1; i++ {
		spaces += " "
	}
	for scanner.Scan() {
		finalData += spaces + scanner.Text() + "\n"
	}
	return finalData
}
