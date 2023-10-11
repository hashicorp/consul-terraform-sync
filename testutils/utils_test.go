// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package testutils

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreePort(t *testing.T) {
	t.Run("ports_are_not_reused", func(t *testing.T) {
		a := FreePort(t)
		b := FreePort(t)

		// wait to ensure listener has freed up port
		time.Sleep(1 * time.Second)
		c := FreePort(t)

		time.Sleep(2 * time.Second)
		d := FreePort(t)

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
		del := MakeTempDir(t, tempDir)

		_, err := os.ReadDir(tempDir)
		require.NoError(t, err)

		del()
		_, err = os.ReadDir(tempDir)
		require.Error(t, err)
	})
}

func TestFindFileMatches(t *testing.T) {
	tempDir := "temp-src"
	del := MakeTempDir(t, tempDir)

	badFiles := []string{filepath.Join(tempDir, "file1.bad"), filepath.Join(tempDir, "file2.bad")}
	goodFiles := []string{filepath.Join(tempDir, "file1.good"), filepath.Join(tempDir, "file2.good")}
	for _, f := range badFiles {
		_, err := os.Create(f)
		require.NoError(t, err)
	}

	for _, f := range goodFiles {
		_, err := os.Create(f)
		require.NoError(t, err)
	}

	mf := FindFileMatches(t, tempDir, "*.good")

	// Sort both slices to make sure the order doesn't affect the equal check
	sort.Strings(goodFiles)
	sort.Strings(mf)
	require.Equal(t, mf, goodFiles)

	del()
}

func TestCopyFiles(t *testing.T) {
	tempDir := "test-temp"
	tempSubDir := filepath.Join(tempDir, "test-sub")
	delRoot := MakeTempDir(t, tempDir)
	del := MakeTempDir(t, tempSubDir)
	tempSrc := "temp-src"
	delSrc := MakeTempDir(t, tempSrc)

	files := []string{filepath.Join(tempSrc, "file1"), filepath.Join(tempSrc, "file2")}
	for _, f := range files {
		_, err := os.Create(f)
		require.NoError(t, err)
	}

	CopyFiles(t, files, tempSubDir)

	for _, f := range files {
		name := filepath.Base(f)
		_, err := os.Stat(filepath.Join(tempSubDir, name))
		require.NoError(t, err)
	}

	del()
	delRoot()
	delSrc()
}

func TestCopyFile(t *testing.T) {
	tempSrc := "test-src"
	delSrc := MakeTempDir(t, tempSrc)

	tempDir := "test-temp"
	delTemp := MakeTempDir(t, tempDir)

	fileName := "file1"
	src := filepath.Join(tempSrc, fileName)
	_, err := os.Create(src)
	require.NoError(t, err)

	dst := filepath.Join(tempDir, fileName)
	CopyFile(t, src, dst)

	_, err = os.Stat(dst)
	require.NoError(t, err)

	delSrc()
	delTemp()
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
