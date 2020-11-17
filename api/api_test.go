package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul-terraform-sync/event"
)

func TestServe(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		path       string
		statusCode int
	}{
		{
			"overall status TODO",
			"status",
			http.StatusNotFound,
		},
		{
			"task status: all",
			"status/tasks",
			http.StatusOK,
		},
		{
			"task status: single",
			"status/tasks/task_b",
			http.StatusOK,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// find a free port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	api := NewAPI(event.NewStore(), port)
	go api.Serve(ctx)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := fmt.Sprintf("http://localhost:%d/%s/%s",
				port, defaultAPIVersion, tc.path)

			resp, err := http.Get(u)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.statusCode, resp.StatusCode)
		})
	}
}

func TestJsonResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		code     int
		response interface{}
	}{
		{
			"task status: error",
			http.StatusBadRequest,
			map[string]string{
				"error": "bad request",
			},
		},
		{
			"task status: success",
			http.StatusOK,
			map[string]TaskStatus{
				"task_a": TaskStatus{
					TaskName:  "task_a",
					Status:    StatusDegraded,
					Providers: []string{"local", "null", "f5"},
					Services:  []string{"api", "web", "db"},
					EventsURL: "/v1/status/tasks/test_task?include=events",
				},
				"task_b": TaskStatus{
					TaskName:  "task_b",
					Status:    StatusUndetermined,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := jsonResponse(w, tc.code, tc.response)
			assert.NoError(t, err)
		})
	}
}
