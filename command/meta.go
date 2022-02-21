package command

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/go-wordwrap"
)

// terminal width. use for word-wrapping
const width = uint(78)

// meta contains the meta-options and functionality for all CTS commands
type meta struct {
	UI    cli.Ui
	flags *flag.FlagSet

	helpOptions []string
	port        *int
	addr        *string

	tls tls
}

type tls struct {
	caPath     *string
	caCert     *string
	clientCert *string
	clientKey  *string
	sslVerify  *bool
}

const (
	// Command line flag names
	FlagPort     = "port"
	FlagHTTPAddr = "http-addr"

	FlagCAPath     = "ca-path"
	FlagCACert     = "ca-cert"
	FlagClientCert = "client-cert"
	FlagClientKey  = "client-key"
	FlagSSLVerify  = "ssl-verify"

	FlagAutoApprove = "auto-approve"
)

func (m *meta) defaultFlagSet(name string) *flag.FlagSet {
	m.flags = flag.NewFlagSet(name, flag.ContinueOnError)

	// Values provide both default values, and documentation for the default value when -help is used
	m.port = m.flags.Int(FlagPort, config.DefaultPort,
		fmt.Sprintf("[Deprecated] The port to use for the Consul-Terraform-Sync API server, "+
			"it is preferred to use the %s field instead.", FlagHTTPAddr))

	m.addr = m.flags.String(FlagHTTPAddr, api.DefaultURL, fmt.Sprintf("The `address` and port of the CTS daemon. The value can be an IP "+
		"address or DNS address, but it must also include the port. This can "+
		"also be specified via the %s environment variable. The "+
		"default value is %s. The scheme can also be set to "+
		"HTTPS by including https in the provided address (eg. https://127.0.0.1:8558)", api.EnvAddress, api.DefaultURL))

	// Initialize TLS flags
	m.tls.caPath = m.flags.String(FlagCAPath, "", fmt.Sprintf("Path to a directory of CA certificates to use for TLS when communicating with Consul-Terraform-Sync. "+
		"This can also be specified using the %s environment variable.", api.EnvTLSCAPath))
	m.tls.caCert = m.flags.String(FlagCACert, "", fmt.Sprintf("Path to a CA file to use for TLS when communicating with Consul-Terraform-Sync. "+
		"This can also be specified using the %s environment variable.", api.EnvTLSCACert))
	m.tls.clientCert = m.flags.String(FlagClientCert, "", fmt.Sprintf("Path to a client cert file to use for TLS when verify_incoming is enabled. "+
		"This can also be specified using the %s environment variable.", api.EnvTLSClientCert))
	m.tls.clientKey = m.flags.String(FlagClientKey, "", fmt.Sprintf("Path to a client key file to use for TLS when verify_incoming is enabled. "+
		"This can also be specified using the %s environment variable.", api.EnvTLSClientKey))
	m.tls.sslVerify = m.flags.Bool(FlagSSLVerify, true, fmt.Sprintf("Boolean to verify SSL or not. Set to true to verify SSL. "+
		"This can also be specified using the %s environment variable.", api.EnvTLSSSLVerify))

	m.flags.SetOutput(ioutil.Discard)

	return m.flags
}

func (m *meta) setHelpOptions() {
	m.flags.VisitAll(func(f *flag.Flag) {
		option := fmt.Sprintf("  %s %s\n    %s\n", f.Name, f.Value, f.Usage)
		m.helpOptions = append(m.helpOptions, option)
	})
	if len(m.helpOptions) == 0 {
		m.helpOptions = append(m.helpOptions, "No options are currently available")
	}
}

func (m *meta) setFlagsUsage(flags *flag.FlagSet, args []string, help string) {
	flags.Usage = func() {
		m.UI.Error(fmt.Sprintf("Error: unsupported arguments in flags '%s'",
			strings.Join(args, ", ")))
		m.UI.Output(fmt.Sprintf("Please see --help information below for "+
			"supported options:\n\n%s", help))
	}
}

func (m *meta) oneArgCheck(name string, args []string) bool {
	numArgs := len(args)
	if numArgs == 1 {
		return true
	}

	m.UI.Error("Error: this command requires one argument: [options] <task name>")
	if numArgs == 0 {
		m.UI.Output("No arguments were passed to the command")
	} else {
		m.UI.Output(fmt.Sprintf("%d arguments were passed to the command: '%s'",
			numArgs, strings.Join(args, ", ")))
		m.UI.Output("All flags are required to appear before positional arguments if set\n")
	}

	help := fmt.Sprintf("For additional help try 'consul-terraform-sync %s --help'",
		name)
	help = wordwrap.WrapString(help, width)

	m.UI.Output(help)
	return false
}

// clientConfig is used to initialize and return a new API ClientConfig using
// the default command line arguments and env vars.
func (m *meta) clientConfig() (*api.ClientConfig, error) {
	// Let the Client determine its default first, then override with command flag values
	c := api.BaseClientConfig()

	// override config values from flags
	if m.isFlagParsedAndFound(FlagPort) {
		m.UI.Warn(fmt.Sprintf("Warning: '%s' option is deprecated and will be removed in a later version. "+
			"It is preferred to use the '%s' option instead.\n", FlagPort, FlagHTTPAddr))

		u, err := url.ParseRequestURI(c.URL)
		if err != nil {
			return nil, err
		}

		u.Host = fmt.Sprintf("%s:%d", u.Hostname(), *m.port) // add port to hostname
		c.URL = u.String()
	}

	if m.isFlagParsedAndFound(FlagHTTPAddr) {
		c.URL = *m.addr
	}

	// If we need custom TLS configuration, then set it
	if m.tls.caCert != nil && *m.tls.caCert != "" {
		c.TLSConfig.CACert = *m.tls.caCert
	}
	if m.tls.caPath != nil && *m.tls.caPath != "" {
		c.TLSConfig.CAPath = *m.tls.caPath
	}
	if m.tls.clientCert != nil && *m.tls.clientCert != "" {
		c.TLSConfig.ClientCert = *m.tls.clientCert
	}
	if m.tls.clientKey != nil && *m.tls.clientKey != "" {
		c.TLSConfig.ClientKey = *m.tls.clientKey
	}
	if m.isFlagParsedAndFound(FlagSSLVerify) {
		c.TLSConfig.SSLVerify = *m.tls.sslVerify
	}

	return c, nil
}

func (m *meta) client() (*api.Client, error) {
	clientConfig, err := m.clientConfig()
	if err != nil {
		return nil, err
	}

	c, err := api.NewClient(clientConfig, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (m *meta) taskLifecycleClient() (*api.TaskLifecycleClient, error) {
	clientConfig, err := m.clientConfig()
	if err != nil {
		return nil, err
	}

	c, err := api.NewTaskLifecycleClient(clientConfig, nil)

	if err != nil {
		return nil, err
	}
	return c, nil
}

// requestUserApproval returns an exit code and boolean describing if the user
// approved a given action. If the user did not approve (false is returned) or
// if there is an error in processing the user input, an exit code is provided.
func (m *meta) requestUserApproval(taskName, action string) (int, bool) {
	m.UI.Output("Only 'yes' will be accepted to approve, enter 'no' or leave blank to reject.\n")
	v, err := m.UI.Ask("Enter a value:")
	m.UI.Output("")

	if err != nil {
		m.UI.Error(fmt.Sprintf("Error asking for approval: %s", err))
		return ExitCodeError, false
	}
	if v != "yes" {
		m.UI.Output(fmt.Sprintf("Cancelled %s task '%s'", action, taskName))
		return ExitCodeOK, false
	}
	return 0, true
}

// requestUserApprovalEnable prints a prompt for user approval of enabling a task
// and waits for the user input. It returns an exit code and boolean describing
// if the user approved.
func (m *meta) requestUserApprovalEnable(taskName string) (int, bool) {
	m.UI.Info("Enabling the task will perform the actions described above.")
	m.terraformApprovalWarning(taskName)
	return m.requestUserApproval(taskName, "enabling")
}

// requestUserApprovalDelete prints a prompt for user approval of deleting a task
// and waits for the user input. It returns an exit code and boolean describing
// if the user approved.
func (m *meta) requestUserApprovalDelete(taskName string) (int, bool) {
	m.UI.Info(fmt.Sprintf("Do you want to delete '%s'?", taskName))
	m.UI.Output(" - This action cannot be undone.")
	m.UI.Output(" - Deleting a task will not destroy the infrastructure managed by the task.")
	m.UI.Output(" - If the task is not running, it will be deleted immediately.")
	m.UI.Output(" - If the task is running, it will be deleted once it has completed.")
	return m.requestUserApproval(taskName, "deleting")
}

// requestUserApprovalCreate prints a prompt for user approval of deleting a task
// and waits for the user input. It returns an exit code and boolean describing
// if the user approved.
func (m *meta) requestUserApprovalCreate(taskName string) (int, bool) {
	m.UI.Info("Creating the task will perform the actions described above.")
	m.terraformApprovalWarning(taskName)
	return m.requestUserApproval(taskName, "creating")
}

// terraformApprovalWarning prints out a standard warning for approving a terraform plan
func (m *meta) terraformApprovalWarning(taskName string) {
	m.UI.Output(fmt.Sprintf("Do you want to perform these actions for '%s'?", taskName))
	m.UI.Output(" - This action cannot be undone.")
	m.UI.Output(" - Consul-Terraform-Sync cannot guarantee Terraform will perform")
	m.UI.Output("   these exact actions if monitored services have changed.\n")
}

// Returns true if the flags have been parsed
// and the flag has been found in the list of provided flags, false otherwise
func (m meta) isFlagParsedAndFound(name string) bool {
	found := false
	m.flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
