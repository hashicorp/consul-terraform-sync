// testutils package contains some helper methods that are used in tests across
// multiple packages

package testutils

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

// FreePort finds the next free port incrementing upwards. Use for testing.
func FreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err = listener.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

// MakeTempDir creates a directory in the current path for a test. Caller is
// responsible for managing the uniqueness of the directory name. Returns a
// function for the caller to delete the temporary directory.
func MakeTempDir(t testing.TB, tempDir string) func() error {
	_, err := os.Stat(tempDir)
	if !os.IsNotExist(err) {
		log.Printf("[WARN] temp dir %s was not cleared out after last test. "+
			"Deleting.", tempDir)
		err = os.RemoveAll(tempDir)
		require.NoError(t, err)
	}
	os.Mkdir(tempDir, os.ModePerm)

	return func() error {
		return os.RemoveAll(tempDir)

	}
}

func DeregisterConsulService(tb testing.TB, srv *testutil.TestServer, id string) {
	u := fmt.Sprintf("http://%s/v1/agent/service/deregister/%s", srv.HTTPAddr, id)
	resp := RequestHTTP(tb, http.MethodPut, u, "")
	defer resp.Body.Close()
}

// RequestHTTP makes an http request. The caller is responsible for closing
// the response.
func RequestHTTP(t testing.TB, method, url, body string) *http.Response {
	r := strings.NewReader(body)
	req, err := http.NewRequest(method, url, r)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// Meets consul/sdk/testutil/TestingTB interface
// Required for any initialization of the test consul server as it requires
// one of these as an argument.
var _ testutil.TestingTB = (*TestingTB)(nil)

// TestingTB implements Consul's testutil.TestingTB
type TestingTB struct {
	cleanup func()
	sync.Mutex
}

// DoCleanup implements Consul's testutil.TestingTB's DoCleanup()
func (t *TestingTB) DoCleanup() {
	t.Lock()
	defer t.Unlock()
	t.cleanup()
}

// Failed implements Consul's testutil.TestingTB's Failed()
func (*TestingTB) Failed() bool { return false }

// Logf implements Consul's testutil.TestingTB's Logf()
func (*TestingTB) Logf(string, ...interface{}) {}

// Name implements Consul's testutil.TestingTB's Name()
func (*TestingTB) Name() string { return "TestingTB" }

// Cleanup implements Consul's testutil.TestingTB's Cleanup()
func (t *TestingTB) Cleanup(f func()) {
	t.Lock()
	defer t.Unlock()
	prev := t.cleanup
	t.cleanup = func() {
		f()
		if prev != nil {
			prev()
		}
	}
}
