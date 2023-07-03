package precommit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/step-security/secure-repo/remediation/workflow/permissions"
	"gopkg.in/yaml.v3"
)

type UpdatePrecommitConfigResponse struct {
	OriginalInput        string
	FinalOutput          string
	IsChanged            bool
	ConfigfileFetchError bool
}

type UpdatePrecommitConfigRequest struct {
	Content   string
	Languages []string
}

type PrecommitConfig struct {
	Repos []Repo `yaml:"repos"`
}

type Repo struct {
	Repo  string `yaml:"repo"`
	Rev   string `yaml:"rev"`
	Hooks []Hook `yaml:"hooks"`
}

type Hook struct {
	Id string `yaml:"id"`
}

type FetchPrecommitConfig struct {
	Hooks Hooks `yaml:"hooks"`
}

type Hooks map[string][]Repo

func getConfigFile() (string, error) {
	filePath := os.Getenv("PRECOMMIT_CONFIG")

	if filePath == "" {
		filePath = "./precommit-config.yml"
	}

	configFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(configFile), nil
}

func GetHooks(precommitConfig string) ([]Repo, error) {
	var updatePrecommitConfigRequest UpdatePrecommitConfigRequest
	json.Unmarshal([]byte(precommitConfig), &updatePrecommitConfigRequest)
	inputConfigFile := []byte(updatePrecommitConfigRequest.Content)
	configMetadata := PrecommitConfig{}
	err := yaml.Unmarshal(inputConfigFile, &configMetadata)
	if err != nil {
		return nil, err
	}

	alreadyPresentHooks := make(map[string]bool)
	for _, repos := range configMetadata.Repos {
		for _, hook := range repos.Hooks {
			alreadyPresentHooks[hook.Id] = true
		}
	}

	configFile, err := getConfigFile()
	if err != nil {
		return nil, err
	}
	var fetchPrecommitConfig FetchPrecommitConfig
	yaml.Unmarshal([]byte(configFile), &fetchPrecommitConfig)
	newHooks := make(map[string]Repo)
	for _, lang := range updatePrecommitConfigRequest.Languages {
		if _, isSupported := fetchPrecommitConfig.Hooks[lang]; !isSupported {
			continue
		}
		if _, ok := alreadyPresentHooks[fetchPrecommitConfig.Hooks[lang][0].Hooks[0].Id]; !ok {
			if repo, ok := newHooks[fetchPrecommitConfig.Hooks[lang][0].Repo]; ok {
				repo.Hooks = append(repo.Hooks, fetchPrecommitConfig.Hooks[lang][0].Hooks...)
				newHooks[fetchPrecommitConfig.Hooks[lang][0].Repo] = repo
			} else {
				newHooks[fetchPrecommitConfig.Hooks[lang][0].Repo] = fetchPrecommitConfig.Hooks[lang][0]
			}
			alreadyPresentHooks[fetchPrecommitConfig.Hooks[lang][0].Hooks[0].Id] = true
		}
	}
	// Adding common hooks
	var repos []Repo
	for _, repo := range fetchPrecommitConfig.Hooks["common"] {
		tempRepo := repo
		tempRepo.Hooks = nil
		hookPresent := false
		for _, hook := range repo.Hooks {
			if _, ok := alreadyPresentHooks[hook.Id]; !ok {
				tempRepo.Hooks = append(tempRepo.Hooks, hook)
				hookPresent = true
			}
		}
		if hookPresent {
			repos = append(repos, tempRepo)
		}
	}
	for _, repo := range newHooks {
		repos = append(repos, repo)
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Repo < repos[j].Repo
	})
	return repos, nil
}

func UpdatePrecommitConfig(precommitConfig string, Hooks []Repo) (*UpdatePrecommitConfigResponse, error) {
	var updatePrecommitConfigRequest UpdatePrecommitConfigRequest
	json.Unmarshal([]byte(precommitConfig), &updatePrecommitConfigRequest)
	inputConfigFile := []byte(updatePrecommitConfigRequest.Content)
	configMetadata := PrecommitConfig{}
	err := yaml.Unmarshal(inputConfigFile, &configMetadata)
	if err != nil {
		return nil, err
	}

	response := new(UpdatePrecommitConfigResponse)
	response.FinalOutput = updatePrecommitConfigRequest.Content
	response.OriginalInput = updatePrecommitConfigRequest.Content
	response.IsChanged = false

	response.FinalOutput = strings.TrimSuffix(response.FinalOutput, "\n")
	repoIndent := 0
	repoGap := 1
	hooksIndent := 2
	hooksGap := 1
	if updatePrecommitConfigRequest.Content == "" {
		response.FinalOutput = "repos:"
	} else {
		repoIndent, repoGap, hooksIndent, hooksGap, err = getPrecommitIndentation(response.FinalOutput)
		if err != nil {
			return nil, err
		}
	}

	for _, Update := range Hooks {
		repoAlreadyExist := false
		for _, update := range configMetadata.Repos {
			if update.Repo == Update.Repo {
				repoAlreadyExist = true
			}
			if repoAlreadyExist {
				break
			}
		}
		response.FinalOutput, err = addHook(Update, repoAlreadyExist, response.FinalOutput,
			repoIndent, repoGap, hooksIndent, hooksGap)
		if err != nil {
			return nil, err
		}
		response.IsChanged = true
	}

	if !strings.HasSuffix(response.FinalOutput, "\n") {
		response.FinalOutput = response.FinalOutput + "\n"
	}

	return response, nil
}

func getPrecommitIndentation(content string) (int, int, int, int, error) {
	lines := strings.Split(content, "\n")

	var repoIndent, repoGap, hooksIndent, hooksGap int
	repoFound, hooksFound := false, false
	for _, line := range lines {
		if strings.Contains(line, "repo:") && !repoFound {
			repoIndent = strings.Index(line, "-")
			repoGap = strings.Index(line, "repo:") - repoIndent - 1
			repoFound = true
		} else if strings.Contains(line, "id:") && !hooksFound {
			hooksIndent = strings.Index(line, "-")
			hooksGap = strings.Index(line, "id:") - hooksIndent - 1
			hooksFound = true
		}

		if repoFound && hooksFound {
			break
		}
	}

	return repoIndent, repoGap, hooksIndent, hooksGap, nil
}

func addHook(Update Repo, repoAlreadyExist bool, inputYaml string, repoIndent, repoGap, hooksIndent, hooksGap int) (string, error) {
	t := yaml.Node{}

	err := yaml.Unmarshal([]byte(inputYaml), &t)
	if err != nil {
		return "", fmt.Errorf("unable to parse yaml %v", err)
	}

	if repoAlreadyExist {
		jobNode := permissions.IterateNode(&t, Update.Repo, "!!str", 0)
		if jobNode == nil {
			return "", fmt.Errorf("Repo Name %s not found in the input yaml", Update.Repo)
		}

		// TODO: Also update rev version for already exist repo
		inputLines := strings.Split(inputYaml, "\n")
		var output []string
		for i := 0; i < jobNode.Line+1; i++ {
			output = append(output, inputLines[i])
		}

		for _, hook := range Update.Hooks {
			hookIndentStr := strings.Repeat(" ", hooksIndent)
			hookGapStr := strings.Repeat(" ", hooksGap)
			output = append(output, fmt.Sprintf("%s-%sid: %s", hookIndentStr, hookGapStr, hook.Id))
		}

		for i := jobNode.Line + 1; i < len(inputLines); i++ {
			output = append(output, inputLines[i])
		}
		return strings.Join(output, "\n"), nil
	} else {
		inputLines := strings.Split(inputYaml, "\n")

		repoIndentStr := strings.Repeat(" ", repoIndent)
		repoGapStr := strings.Repeat(" ", repoGap)
		inputLines = append(inputLines, fmt.Sprintf("%s-%srepo: %s", repoIndentStr, repoGapStr, Update.Repo))

		revIndentStr := strings.Repeat(" ", repoIndent+repoGap+1)
		inputLines = append(inputLines, fmt.Sprintf("%srev: %s", revIndentStr, Update.Rev))

		inputLines = append(inputLines, fmt.Sprintf("%shooks:", revIndentStr))

		hookIndentStr := strings.Repeat(" ", hooksIndent)
		hookGapStr := strings.Repeat(" ", hooksGap)
		for _, hook := range Update.Hooks {
			inputLines = append(inputLines, fmt.Sprintf("%s-%sid: %s", hookIndentStr, hookGapStr, hook.Id))
		}

		return strings.Join(inputLines, "\n"), nil
	}
}
