// testutils package contains some helper methods that are used in tests across
// multiple packages

package testutils

import (
	"log"
	"os"
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
