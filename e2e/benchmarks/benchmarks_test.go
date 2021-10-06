// +build e2e

package benchmarks

import (
	"io/ioutil"

	"github.com/hashicorp/go-hclog"
)

func init() {
	// Mutes CTS logging when run directly via CLI or controller
	hclog.SetDefault(hclog.New(&hclog.LoggerOptions{
		Output: ioutil.Discard,
	}))
}
