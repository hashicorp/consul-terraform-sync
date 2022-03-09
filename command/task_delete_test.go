package command

import (
	"flag"
	"fmt"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/posener/complete"
	"github.com/stretchr/testify/assert"
)

func TestTaskDelete_AutocompleteFlags(t *testing.T) {
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

func TestTaskDelete_AutocompleteArgs(t *testing.T) {
	cmd := newTaskDisableCommand(meta{UI: cli.NewMockUi()})
	c := cmd.AutocompleteArgs()
	assert.Equal(t, complete.PredictNothing, c)
}
