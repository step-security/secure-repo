package oidc_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/step-security/secure-repo/remediation/workflow/oidc"
)

const inputDirectory = "../../../testfiles/OIDC/input"
const outputDirectory = "../../../testfiles/OIDC/output"

func TestUpdateActionsToUseOIDC(t *testing.T) {
	inputFiles := []string{
		"withTopLevelAndJobLevel.yml",
		"withoutTopLevel.yml",
		"withoutAnyPerms.yml",
		"multipleAWS.yml",
		"multipleAWSSteps.yml",
	}
	os.Setenv("KBFolder", "../../../knowledge-base/actions")

	for _, inputFile := range inputFiles {
		t.Run(inputFile, func(t *testing.T) {
			inputPath := filepath.Join(inputDirectory, inputFile)
			expectedOutputPath := filepath.Join(outputDirectory, inputFile)

			inputYAML, err := ioutil.ReadFile(inputPath)
			if err != nil {
				t.Errorf("Failed to read input YAML file: %v", err)
				return
			}

			expectedOutputYAML, err := ioutil.ReadFile(expectedOutputPath)
			if err != nil {
				t.Errorf("Failed to read expected output YAML file: %v", err)
				return
			}

			updatedYAML, _, err := oidc.UpdateActionsToUseOIDC(string(inputYAML))
			if err != nil {
				t.Errorf("Failed to update actions to use OIDC: %v", err)
				return
			}

			if updatedYAML != string(expectedOutputYAML) {
				t.Errorf("Updated YAML does not match the expected output:\nExpected:\n%s\n\nActual:\n%s", string(expectedOutputYAML), updatedYAML)
			}
		})
	}
}
