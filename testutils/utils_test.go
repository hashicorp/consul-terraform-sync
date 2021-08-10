package testutils

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreePort(t *testing.T) {
	t.Run("ports_are_not_reused", func(t *testing.T) {
		a, err := FreePort()
		require.NoError(t, err)
		b, err := FreePort()
		require.NoError(t, err)

		// wait to ensure listener has freed up port
		time.Sleep(1 * time.Second)
		c, err := FreePort()
		require.NoError(t, err)

		time.Sleep(2 * time.Second)
		d, err := FreePort()
		require.NoError(t, err)

		assert.NotEqual(t, a, b)
		assert.NotEqual(t, a, c)
		assert.NotEqual(t, a, d)
		assert.NotEqual(t, b, c)
		assert.NotEqual(t, b, d)
		assert.NotEqual(t, c, d)
	})
}

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

func TestSetEnvVar(t *testing.T) {
	envvar := "CTS_TEST_ENV_VAR"
	t.Run("env var with existing value", func(t *testing.T) {
		// set up environment variable with a value
		os.Setenv(envvar, "old-value")
		defer os.Unsetenv(envvar)

		// check that setting worked
		reset := Setenv(envvar, "new-value")
		actual, ok := os.LookupEnv(envvar)
		require.True(t, ok)
		assert.Equal(t, "new-value", actual)

		// check resetting back to original value
		reset()
		actual, ok = os.LookupEnv(envvar)
		require.True(t, ok)
		assert.Equal(t, "old-value", actual)
	})
	t.Run("env var with no value", func(t *testing.T) {
		// check that setting worked
		reset := Setenv(envvar, "value")
		actual, ok := os.LookupEnv(envvar)
		assert.True(t, ok)
		assert.Equal(t, "value", actual)

		// check resetting back to no value
		reset()
		actual, ok = os.LookupEnv(envvar)
		assert.False(t, ok)
		assert.Empty(t, actual)
	})
}
