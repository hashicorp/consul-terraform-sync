package command

import (
	"errors"
	"flag"
	"fmt"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/api"
	"github.com/mitchellh/cli"
	"github.com/posener/complete"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTaskEnableCommand_AutocompleteFlags(t *testing.T) {
	t.Parallel()
	cmd := newTaskDisableCommand(meta{UI: cli.NewMockUi()})

	predictor := cmd.AutocompleteFlags()

	// Test that we get the expected number of predictions
	args := complete.Args{Last: "-"}
	res := predictor.Predict(args)

	// Grab the list of flags from the Flag object
	flags := make([]string, 0)
	cmd.flags.VisitAll(func(flag *flag.Flag) {
		flags = append(flags, fmt.Sprintf("-%s", flag.Name))
	})

	// Verify that there is a prediction for each flag associated with the command
	assert.Equal(t, len(flags), len(res))
	assert.ElementsMatch(t, flags, res, "flags and predictions didn't match, make sure to add "+
		"new flags to the command AutoCompleteFlags function")
}

func TestTaskEnableCommand_AutocompleteArgs(t *testing.T) {

	cases := []struct {
		name         string
		enabledNames []string
		taskStatus   map[string]bool
	}{
		{
			name:         "nominal",
			enabledNames: []string{"first", "second", "third"},
			taskStatus: map[string]bool{
				"first":  false,
				"second": false,
				"third":  false,
				"fourth": true,
			},
		},
		{
			name:         "no tasks",
			enabledNames: []string{},
			taskStatus:   make(map[string]bool),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newTaskEnableCommand(meta{UI: cli.NewMockUi()})

			p := new(mocks.ClientWithResponsesInterface)
			cmd.predictorClient = p

			tasks := make([]oapigen.Task, 0, len(tc.taskStatus))
			for k, v := range tc.taskStatus {
				enabled := v
				task := oapigen.Task{
					Name:    k,
					Enabled: &enabled,
				}
				tasks = append(tasks, task)
			}

			tasksResponse := oapigen.TasksResponse{
				RequestId: "!@#$%^&*()?!abc",
				Tasks:     &tasks,
			}

			resp := oapigen.GetAllTasksResponse{
				JSON200: &tasksResponse,
			}

			// Return the response, and expect only enabled task names to be present in the prediction
			p.On("GetAllTasksWithResponse", mock.Anything).Return(&resp, nil)

			predictor := cmd.AutocompleteArgs()

			res := predictor.Predict(complete.Args{})

			assert.ElementsMatch(t, tc.enabledNames, res, "flags and predictions didn't match, make sure to add "+
				"new flags to the command AutoCompleteFlags function")
		})
	}
}

func TestTaskEnableCommand_AutocompleteArgs_Errors(t *testing.T) {

	scenarioClientError := "client error"
	scenarioEmptyTasks := "empty tasks"

	cases := []struct {
		name     string
		scenario string
	}{
		{
			name:     "predictor client returns error",
			scenario: scenarioClientError,
		},
		{
			name:     "empty task response",
			scenario: scenarioEmptyTasks,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newTaskEnableCommand(meta{UI: cli.NewMockUi()})

			p := new(mocks.ClientWithResponsesInterface)
			cmd.predictorClient = p

			switch tc.scenario {
			case scenarioClientError:
				err := errors.New("some error")
				p.On("GetAllTasksWithResponse", mock.Anything).Return(nil, err)
			case scenarioEmptyTasks:
				resp := oapigen.GetAllTasksResponse{}
				p.On("GetAllTasksWithResponse", mock.Anything).Return(&resp, nil)
			}

			predictor := cmd.AutocompleteArgs()

			// Not panicking is a success
			predictor.Predict(complete.Args{})
		})
	}
}
