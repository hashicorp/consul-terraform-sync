package command

// TaskDisableCommand handles the `task disable` command
type taskDisableCommand struct {
	meta
}

// Help returns the command's usage, list of flags, and examples
func (c *taskDisableCommand) Help() string {
	return "TODO"
}

// Synopsis is a short one-line synopsis of the command
func (c *taskDisableCommand) Synopsis() string {
	return "Disables existing tasks from running."
}

// Run runs the command
func (c *taskDisableCommand) Run(args []string) int {
	c.UI.Info("Running...TODO")
	return 0
}
