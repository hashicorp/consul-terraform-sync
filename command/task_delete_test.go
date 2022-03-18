package command

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/api"
	"github.com/mitchellh/cli"
	"github.com/posener/complete"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTaskDeleteCommand_AutocompleteFlags(t *testing.T) {
	t.Parallel()
	cmd := newTaskDeleteCommand(meta{UI: cli.NewMockUi()})

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

func TestTaskDeleteCommand_AutocompleteArgs(t *testing.T) {

	cases := []struct {
		name      string
		taskNames []string
	}{
		{
			name:      "nominal",
			taskNames: []string{"first", "second", "third"},
		},
		{
			name:      "no tasks",
			taskNames: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newTaskDeleteCommand(meta{UI: cli.NewMockUi()})

			p := new(mocks.ClientInterface)
			cmd.predictorClient = p

			tasks := make([]oapigen.Task, len(tc.taskNames))
			for i, n := range tc.taskNames {
				tasks[i].Name = n
			}

			tasksResponse := api.TasksResponse{
				RequestId: "!@#$%^&*()?!abc",
				Tasks:     &tasks,
			}

			// Marshal tasksResponse, and create new readerCloser
			b, err := json.Marshal(&tasksResponse)
			require.NoError(t, err)

			r := bytes.NewReader(b)
			rc := ioutil.NopCloser(r)

			var resp http.Response
			resp.Body = rc

			// Return the response, and expect each task name to be present in the prediction
			p.On("GetTasks", mock.Anything).Return(&resp, nil)

			predictor := cmd.AutocompleteArgs()

			res := predictor.Predict(complete.Args{})

			assert.ElementsMatch(t, tc.taskNames, res, "flags and predictions didn't match, make sure to add "+
				"new flags to the command AutoCompleteFlags function")
		})
	}
}

func TestTaskDeleteCommand_AutocompleteArgs_Errors(t *testing.T) {

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
			cmd := newTaskDeleteCommand(meta{UI: cli.NewMockUi()})

			p := new(mocks.ClientInterface)
			cmd.predictorClient = p

			switch tc.scenario {
			case scenarioClientError:
				err := errors.New("some error")
				p.On("GetTasks", mock.Anything).Return(nil, err)
			case scenarioEmptyTasks:
				// Marshal tasksResponse, and create new readerCloser
				b, err := json.Marshal(&api.TaskResponse{})
				require.NoError(t, err)

				r := bytes.NewReader(b)
				rc := ioutil.NopCloser(r)

				var resp http.Response
				resp.Body = rc
				p.On("GetTasks", mock.Anything).Return(&resp, err)
			}

			predictor := cmd.AutocompleteArgs()

			// Not panicking is a success
			predictor.Predict(complete.Args{})
		})
	}
}
