package workflow

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func Test_AddWorkflow(t *testing.T) {
	tests := []struct {
		workflowName       string
		workflowParameters WorkflowParameters
		expectedError      bool
		expectedOutputFile string
	}{
		{
			workflowName: "CodeQL",
			workflowParameters: WorkflowParameters{
				LanguagesToAdd: []string{"cpp", "go", "java"},
				DefaultBranch:  "main",
			},
			expectedError:      false,
			expectedOutputFile: "../../testfiles/addworkflow/expected-codeql.yml",
		},
		{
			workflowName: "xyz",
			workflowParameters: WorkflowParameters{
				LanguagesToAdd: []string{"cpp"},
				DefaultBranch:  "main",
			},
			expectedError:      true,
			expectedOutputFile: "",
		},
		{
			workflowName:       "Dependency-review",
			workflowParameters: WorkflowParameters{},
			expectedError:      false,
			expectedOutputFile: "../../testfiles/addworkflow/expected-dependency-review.yml",
		},
		{
			workflowName: "Scorecard",
			workflowParameters: WorkflowParameters{
				DefaultBranch: "main",
			},
			expectedError:      false,
			expectedOutputFile: "../../testfiles/addworkflow/expected-scorecards.yml",
		},
	}

	for _, test := range tests {
		output, err := AddWorkflow(test.workflowName, test.workflowParameters)
		if err != nil {
			if !test.expectedError {
				t.Errorf("Error adding Workflow: %v", err)
			}
			continue
		}
		expectedOutput, err := ioutil.ReadFile(test.expectedOutputFile)
		if err != nil {
			t.Errorf("Error in reading file: %v", err)
		}

		if !reflect.DeepEqual(output, string(expectedOutput)) {
			t.Errorf("test failed %s did not match expected output\n%s", test.workflowName, output)
		}
	}

}
