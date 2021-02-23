// testutils package contains some helper methods that are used in tests across
// multiple packages

package testutils

import (
	"log"
	"os"
	"sync"

	"github.com/hashicorp/consul/sdk/testutil"
)

// MakeTempDir creates a directory in the current path for a test. Caller is
// responsible for managing the uniqueness of the directory name. Returns a
// function for the caller to delete the temporary directory.
func MakeTempDir(tempDir string) (func() error, error) {
	_, err := os.Stat(tempDir)
	if !os.IsNotExist(err) {
		log.Printf("[WARN] temp dir %s was not cleared out after last test. "+
			"Deleting.", tempDir)
		if err = os.RemoveAll(tempDir); err != nil {
			return nil, err
		}
	}
	os.Mkdir(tempDir, os.ModePerm)

	return func() error {
		return os.RemoveAll(tempDir)

	}, nil
}

// Meets consul/sdk/testutil/TestingTB interface
// Required for any initialization of the test consul server as it requires
// one of these as an argument.
var _ testutil.TestingTB = (*TestingTB)(nil)

type TestingTB struct {
	cleanup func()
	sync.Mutex
}

func (t *TestingTB) DoCleanup() {
	t.Lock()
	defer t.Unlock()
	t.cleanup()
}

func (*TestingTB) Failed() bool                { return false }
func (*TestingTB) Logf(string, ...interface{}) {}
func (*TestingTB) Name() string                { return "TestingTB" }
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
