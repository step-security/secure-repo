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
			workflowName: "codeql",
			workflowParameters: WorkflowParameters{
				languagesToAdd: []string{"cpp", "go", "java"},
				defaultBranch:  "main",
			},
			expectedError:      false,
			expectedOutputFile: "../../testfiles/expected-codeql.yml",
		},
		{
			workflowName: "xyz",
			workflowParameters: WorkflowParameters{
				languagesToAdd: []string{"cpp"},
				defaultBranch:  "main",
			},
			expectedError:      true,
			expectedOutputFile: "",
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
