package command

import (
	"flag"
	"fmt"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/posener/complete"
	"github.com/stretchr/testify/assert"
)

func TestStartCommand_Name(t *testing.T) {
	cmd := newStartCommand(meta{UI: cli.NewMockUi()}, false)
	assert.Equal(t, cmdStartName, cmd.Name())
}

func TestStartCommand_Help(t *testing.T) {
	cmd := newStartCommand(meta{UI: cli.NewMockUi()}, false)

	contains := []string{
		"Usage CLI: consul-terraform-sync start [-help] [options]",
		"Options:",
		"-config-dir",
		"-config-file",
		"-inspect",
		"-inspect-task",
		"-once",
	}

	doesNotContain := []string{
		"-autocomplete-install",
		"-autocomplete-uninstall",
	}

	s := cmd.Help()
	for _, c := range contains {
		assert.Contains(t, s, c)
	}

	for _, c := range doesNotContain {
		assert.NotContains(t, s, c)
	}
}

func TestStartCommand_HelpDefault(t *testing.T) {
	cmd := newStartCommand(meta{UI: cli.NewMockUi()}, false)

	contains := []string{
		"Usage CLI: consul-terraform-sync <command> [-help] [options]",
		"Options:",
		"-autocomplete-install",
		"-autocomplete-uninstall",
		"-config-dir",
		"-config-file",
		"-inspect",
		"-inspect-task",
		"-once",
	}

	s := cmd.HelpDefault()
	for _, c := range contains {
		assert.Contains(t, s, c)
	}
}

func TestStartCommand_Synopsis(t *testing.T) {
	cmd := newStartCommand(meta{UI: cli.NewMockUi()}, false)
	assert.Equal(t, "", cmd.Synopsis())
}

func TestStartCommand_AutocompleteFlags(t *testing.T) {
	t.Parallel()
	cmd := newStartCommand(meta{UI: cli.NewMockUi()}, false)

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

func TestStartCommand_AutocompleteArgs(t *testing.T) {
	cmd := newStartCommand(meta{UI: cli.NewMockUi()}, false)
	c := cmd.AutocompleteArgs()
	assert.Equal(t, complete.PredictNothing, c)
}
