package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/driver"
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
			h := newOverallStatusHandler(event.NewStore(), nil, tc.version)
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
			OverallStatus{
				TaskSummary: TaskSummary{
					Status: StatusSummary{
						Successful: 2,
						Errored:    1,
						Critical:   1,
						Unknown:    1,
					},
					Enabled: EnabledSummary{
						True:  4,
						False: 1,
					},
				},
			},
		},
	}

	// set up store and handler
	store := event.NewStore()
	eventsA := createTaskEvents("success_a", []bool{true})
	addEvents(store, eventsA)
	eventsB := createTaskEvents("success_b", []bool{true, true})
	addEvents(store, eventsB)
	eventsC := createTaskEvents("errored_c", []bool{false, true})
	addEvents(store, eventsC)
	eventsD := createTaskEvents("critical_d", []bool{false, false, true})
	addEvents(store, eventsD)

	// set up driver
	drivers := driver.NewDrivers()
	drivers.Add("success_a", createDriver("success_a", true))
	drivers.Add("success_b", createDriver("success_b", true))
	drivers.Add("errored_c", createDriver("errored_c", true))
	drivers.Add("critical_d", createDriver("critical_d", true))
	drivers.Add("disabled_e", createDriver("disabled_e", false))

	handler := newOverallStatusHandler(store, drivers, "v1")

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
