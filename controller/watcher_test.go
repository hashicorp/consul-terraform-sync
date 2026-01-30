// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcat/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchRetry_retryConsul(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		maxRetry        int
		expectedRetries int
		breakLimit      int
	}{
		{
			name:            "happy path: retry 10x",
			maxRetry:        10,
			expectedRetries: 10,
			breakLimit:      20,
		},
		{
			name:            "happy path: indefinite retry",
			maxRetry:        -1,
			breakLimit:      8,
			expectedRetries: 8,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wr := watcherRetry{
				maxRetries: tc.maxRetry,
				waitFunc: func(attempt int, random *rand.Rand, maxWaitTime time.Duration) time.Duration {
					return 1 * time.Nanosecond
				},
			}

			count := 0
			for true {
				isSuccessRetry, _ := wr.retryConsul(count)
				if !isSuccessRetry || count > tc.breakLimit {
					break
				}
				count++
			}

			// count == number of tries
			// retries are tries - 1, therefore check for count-1 retries
			assert.Equal(t, tc.expectedRetries, count-1)
		})
	}
}

func Test_newWatcherEventHandler(t *testing.T) {
	toJson := func(v any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}
	toGo := func(v any) string {
		return fmt.Sprintf("%+v", v) // Go struct format
	}

	testCases := []struct {
		logLevel   string
		shouldLog  bool
		findString func(v any) string
		event      events.Event
	}{
		{"TRACE", true, toJson, events.Trace{ID: "logged", Message: ""}},
		{"DEBUG", false, toGo, events.Trace{ID: "not logged"}},
		{"TRACE", true, toJson, events.NoNewData{ID: "logged"}},
		{"DEBUG", true, toGo, events.NewData{ID: "logged"}},
		{"INFO", false, toGo, events.RetryAttempt{ID: "not logged"}},
		{"DEBUG", true, func(v any) string { return "type=nil" }, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.logLevel, func(t *testing.T) {
			var buf bytes.Buffer
			logger := logging.NewTestLogger(tc.logLevel, &buf)
			handler := newWatcherEventHandler(logger)
			handler(tc.event)

			logOutput := buf.String()
			fmt.Println("Log Output:", logOutput) // Debugging

			toFind := tc.findString(tc.event)

			if tc.shouldLog {
				// Check if `toFind` is JSON
				var jsonObj map[string]any
				if err := json.Unmarshal([]byte(toFind), &jsonObj); err == nil {
					// If JSON, extract the JSON part from the log and compare
					matches := regexp.MustCompile(`event="({.*})"`).FindStringSubmatch(logOutput)
					require.NotNil(t, matches, "Log output did not contain JSON event")

					jsonStr, err := strconv.Unquote(`"` + matches[1] + `"`)
					require.NoError(t, err, "Failed to unescape JSON string")

					assert.JSONEq(t, toFind, jsonStr, "Extracted JSON does not match expected")
				} else {
					// Otherwise, check if the Go struct format is present in the log
					assert.Contains(t, logOutput, toFind, "Expected non-JSON event representation in log")
				}
			} else {
				assert.Empty(t, logOutput, "Expected no log output, but got: %s", logOutput)
			}
		})
	}
}
