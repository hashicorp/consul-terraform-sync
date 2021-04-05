// testutils package contains some helper methods that are used in tests across
// multiple packages

package testutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/testutils/sdk"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

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

// CheckDir checks whether or not a directory exists. If it exists, returns the
// file infos for further checking.
func CheckDir(t testing.TB, exists bool, dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if exists {
		require.NoError(t, err)
		return files
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	return []os.FileInfo{}
}

// WriteFile write a content to a file path.
func WriteFile(t testing.TB, path, content string) {
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write([]byte(content))
	require.NoError(t, err)
}

// RegisterConsulService regsiters a service to the Consul Catalog. The Consul
// sdk/testutil package currently does not support a method to register multiple
// service instances, distinguished by their IDs.
func RegisterConsulService(tb testing.TB, srv *testutil.TestServer,
	s testutil.TestService, health string) {

	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	require.NoError(tb, enc.Encode(&s))

	u := fmt.Sprintf("http://%s/v1/agent/service/register", srv.HTTPAddr)
	resp := RequestHTTP(tb, http.MethodPut, u, body.String())
	defer resp.Body.Close()

	sdk.AddCheck(srv, tb, s.ID, s.ID, testutil.HealthPassing)
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
