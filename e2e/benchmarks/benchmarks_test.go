//go:build e2e
// +build e2e

package benchmarks

import (
	"io/ioutil"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func init() {
	// Mutes CTS logging when run directly via CLI or controller
	hclog.SetDefault(hclog.New(&hclog.LoggerOptions{
		Output: ioutil.Discard,
	}))
}

// TestBenchmarks_Compile confirms that the benchmark tests are compilable.
// Benchmark tests are only run weekly. This test is intended to run with each
// change (vs. weekly) to do a basic check that the tests are still in a
// compilable state.
func TestBenchmarks_Compile(t *testing.T) {
	// no-op
}
