// testutils package contains some helper methods that are used in tests across
// multiple packages

package testutils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

// FreePort finds the next free port incrementing upwards. Use for testing.
func FreePort(t testing.TB) int {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port
	err = listener.Close()
	require.NoError(t, err)

	return port
}

// MakeTempDir creates a directory in the current path for a test. Caller is
// responsible for managing the uniqueness of the directory name. Returns a
// function for the caller to delete the temporary directory.
func MakeTempDir(t testing.TB, tempDir string) func() error {
	_, err := os.Stat(tempDir)
	if !os.IsNotExist(err) {
		logging.Global().Warn("temp dir was not cleared out after last test. "+
			"Deleting.", "temp_dir", tempDir)
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

// CheckFile checks whether a file exists or not. If it exists, returns the
// contents for further checking. If path parameter already includes filename,
// leave filename parameter as an empty string.
func CheckFile(t testing.TB, exists bool, path, filename string) string {
	fp := filepath.Join(path, filename) // handles if filename is empty
	// Check if file exists
	_, err := os.Stat(fp)
	if !exists {
		require.Error(t, err, fmt.Sprintf("file '%s' is not supposed to exist", filename))
		require.True(t, errors.Is(err, os.ErrNotExist),
			fmt.Sprintf("unexpected error when file '%s' is not supposed to exist", filename))
		return ""
	}
	require.NoError(t, err, fmt.Sprintf("file '%s' does not exist", filename))

	// Return content of file if exists
	content, err := ioutil.ReadFile(fp)
	require.NoError(t, err, fmt.Sprintf("unable to read file '%s'", filename))
	return string(content)
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

// Setenv sets an environment variable to a value. Returns a reset function to
// reset the environment variable back to the original state.
func Setenv(envvar, value string) func() {
	original, ok := os.LookupEnv(envvar)
	os.Setenv(envvar, value)

	return func() {
		if ok {
			os.Setenv(envvar, original)
		} else {
			os.Unsetenv(envvar)
		}
	}
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
