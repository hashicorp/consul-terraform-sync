// testutils package contains some helper methods that are used in tests across
// multiple packages

package testutils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/hashicorp/go-rootcerts"
	"github.com/stretchr/testify/require"
)

type TLSConfig struct {
	ClientCert     string
	ClientKey      string
	CACert         string
	CAFile         string
	VerifyIncoming bool
}

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

// FindFileMatches walks a root directory and returns a list of all files that match
// a particular pattern string.
// eg. If you want to find all files that end with .txt, pattern=*.txt
func FindFileMatches(t testing.TB, rootDir, pattern string) []string {
	var matches []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			require.NoError(t, err)
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	require.NoError(t, err)
	return matches
}

// CopyFiles copies a list of files to a destination (dst) directory
func CopyFiles(t testing.TB, srcFiles []string, dst string) {
	for _, src := range srcFiles {
		file := filepath.Base(src)
		dst := filepath.Join(dst, file)
		CopyFile(t, src, dst)
	}
}

// CopyFile copies a file from src to dst.
func CopyFile(t testing.TB, src, dst string) {
	sourceFI, err := os.Stat(src)
	require.NoError(t, err)
	if !sourceFI.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		err = fmt.Errorf("non-regular source file %s (%q)", sourceFI.Name(), sourceFI.Mode().String())
		require.NoError(t, err)
	}

	destFI, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			require.NoError(t, err)
		}
	} else {
		if !(destFI.Mode().IsRegular()) {
			err = fmt.Errorf("non-regular destination file %s (%q)", destFI.Name(), destFI.Mode().String())
			require.NoError(t, err)
		}
		if os.SameFile(sourceFI, destFI) {
			return
		}
	}

	err = copyFileContents(src, dst)
	require.NoError(t, err)
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

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// RequestHTTPS makes an https request using TLS. The caller is responsible for closing
// the response.
func RequestHTTPS(t testing.TB, method, url, body string, conf TLSConfig) *http.Response {
	r := strings.NewReader(body)
	req, err := http.NewRequest(method, url, r)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	tlsClientConfig := &tls.Config{
		InsecureSkipVerify: !conf.VerifyIncoming,
	}

	tlsCert, err := tls.LoadX509KeyPair(conf.ClientCert, conf.ClientKey)
	require.NoError(t, err)
	tlsClientConfig.Certificates = []tls.Certificate{tlsCert}

	rootConfig := &rootcerts.Config{
		CAFile: conf.CAFile,
	}

	err = rootcerts.ConfigureTLS(tlsClientConfig, rootConfig)
	require.NoError(t, err)

	h := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsClientConfig}}

	resp, err := h.Do(req)
	require.NoError(t, err)
	return resp
}

// RequestJSON encodes the body to JSON and makes an HTTP request. The caller is
// responsible for closing the response.
func RequestJSON(t testing.TB, method, url string, body interface{}) *http.Response {
	// Encode request body
	var r bytes.Buffer
	enc := json.NewEncoder(&r)
	err := enc.Encode(body)
	require.NoError(t, err)

	// Make request
	req, err := http.NewRequest(method, url, &r)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
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

// Fatalf implements Consul's testutil.TestingTB's Fatalf()
func (*TestingTB) Fatalf(string, ...interface{}) {}

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

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// HttpIntercept represents a mapping of a path to the response that the
// HTTP client should return for requests to that path (including query
// parameters).
type HttpIntercept struct {
	Path               string
	RequestTest        func(*testing.T, *http.Request)
	ResponseStatusCode int
	ResponseData       []byte
}

// NewHttpClient returns an HTTP client that returns mocked responses to specific
// request paths based on the specified intercepts.
func NewHttpClient(t *testing.T, intercepts []*HttpIntercept) *http.Client {
	f := func(req *http.Request) *http.Response {
		path := req.URL.Path
		query := req.URL.RawQuery
		if query != "" {
			path = fmt.Sprintf("%s?%s", path, query)
		}
		intercept, err := find(intercepts, path)
		if err != nil {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBufferString(err.Error())),
			}
		}

		if intercept.RequestTest != nil {
			intercept.RequestTest(t, req)
		}

		return &http.Response{
			StatusCode: intercept.ResponseStatusCode,
			Body:       io.NopCloser(bytes.NewBuffer(intercept.ResponseData)),
		}
	}

	return &http.Client{Transport: roundTripFunc(f)}
}

func find(intercepts []*HttpIntercept, path string) (*HttpIntercept, error) {
	for _, c := range intercepts {
		if c.Path == path {
			return c, nil
		}
	}

	return nil, fmt.Errorf("no intercept found for path [%s]", path)
}
