package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverallStatus_New(t *testing.T) {
	cases := []struct {
		name    string
		version string
	}{
		{
			"happy path",
			"v1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newOverallStatusHandler(event.NewStore(), tc.version)
			assert.Equal(t, tc.version, h.version)
		})
	}
}

func TestOverallStatus_ServeHTTP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		path       string
		statusCode int
		expected   OverallStatus
	}{
		{
			"happy path",
			"/v1/status",
			http.StatusOK,
			OverallStatus{Status: StatusErrored},
		},
	}

	// set up store and handler
	store := event.NewStore()
	eventA := event.Event{TaskName: "task_a", Success: true}
	store.Add(eventA)
	eventB := event.Event{TaskName: "task_b", Success: false}
	store.Add(eventB)

	handler := newOverallStatusHandler(store, "v1")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.path, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			require.Equal(t, tc.statusCode, resp.Code)
			if tc.statusCode != http.StatusOK {
				return
			}

			decoder := json.NewDecoder(resp.Body)
			var actual OverallStatus
			err = decoder.Decode(&actual)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestOverallStatus_TaskStatusToOverall(t *testing.T) {
	cases := []struct {
		name     string
		statuses []string
		expected string
	}{
		{
			"successful: all successful",
			[]string{StatusSuccessful, StatusSuccessful, StatusSuccessful,
				StatusSuccessful, StatusSuccessful},
			StatusSuccessful,
		},
		{
			"errored: mix of errored and successful",
			[]string{StatusSuccessful, StatusSuccessful, StatusErrored,
				StatusSuccessful, StatusErrored},
			StatusErrored,
		},
		{
			"critical: at least one critical",
			[]string{StatusSuccessful, StatusSuccessful, StatusErrored,
				StatusSuccessful, StatusCritical},
			StatusCritical,
		},
		{
			"no data",
			[]string{},
			StatusUnknown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := taskStatusToOverall(tc.statuses)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
