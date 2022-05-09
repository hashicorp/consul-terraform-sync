package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/server"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
			h := newOverallStatusHandler(new(mocks.Server), tc.version)
			assert.Equal(t, tc.version, h.version)
		})
	}
}

func TestOverallStatus_ServeHTTP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		path       string
		method     string
		statusCode int
		expected   OverallStatus
	}{
		{
			"happy path",
			"/v1/status",
			http.MethodGet,
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
		{
			"method not allowed",
			"/v1/status",
			http.MethodPatch,
			http.StatusMethodNotAllowed,
			OverallStatus{},
		},
	}

	ctrl := new(mocks.Server)
	taskSetup := map[string]bool{
		"success_a":  true,
		"success_b":  true,
		"errored_c":  true,
		"critical_d": true,
		"disabled_e": false,
	}
	confs := make(config.TaskConfigs, 0, len(taskSetup))
	for taskName, enabled := range taskSetup {
		conf := createTaskConf(taskName, enabled)
		confs = append(confs, &conf)
	}

	events := map[string][]event.Event{
		"success_a":  {{Success: true}},
		"success_b":  {{Success: true}, {Success: true}},
		"errored_c":  {{Success: false}, {Success: true}},
		"critical_d": {{Success: false}, {Success: false}, {Success: true}},
	}
	ctrl.On("Events", mock.Anything, "").Return(events, nil).
		On("Tasks", mock.Anything).Return(confs)

	handler := newOverallStatusHandler(ctrl, "v1")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
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
