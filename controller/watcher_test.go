package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
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
		return fmt.Sprintf("%+v", v)
	}

	testCases := []struct {
		logLevel   string
		shouldLog  bool
		findString func(v any) string
		event      events.Event
	}{
		{"TRACE", true, toJson, events.Trace{ID: "logged"}},
		{"DEBUG", false, toGo, events.Trace{ID: "not logged"}},
		{"TRACE", true, toJson, events.NoNewData{ID: "logged"}},
		{"DEBUG", true, toGo, events.NewData{ID: "logged"}},
		{"INFO", false, toGo, events.RetryAttempt{ID: "not logged"}},
		{"DEBUG", true, func(v any) string { return "type=nil" }, nil},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		logger := logging.NewTestLogger(tc.logLevel, &buf)
		handler := newWatcherEventHandler(logger)
		handler(tc.event)
		toFind := tc.findString(tc.event)
		if tc.shouldLog {
			assert.Contains(t, buf.String(), toFind)
		} else {
			assert.NotContains(t, buf.String(), toFind)
			assert.Equal(t, "", buf.String())
		}
	}
}
