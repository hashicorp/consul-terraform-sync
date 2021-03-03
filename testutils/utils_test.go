package testutils

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeTempDir(t *testing.T) {
	t.Run("happy-path", func(t *testing.T) {
		tempDir := "test-temp"
		delete := MakeTempDir(t, tempDir)

		_, err := ioutil.ReadDir(tempDir)
		require.NoError(t, err)

		delete()
		_, err = ioutil.ReadDir(tempDir)
		require.Error(t, err)
	})
}
