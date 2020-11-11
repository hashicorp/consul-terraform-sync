package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
			"overall status",
			"status",
			http.StatusOK,
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

	port, err := FreePort()
	require.NoError(t, err)
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

func TestServe_context_cancel(t *testing.T) {
	t.Parallel()

	port, err := FreePort()
	require.NoError(t, err)
	api := NewAPI(event.NewStore(), port)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := api.Serve(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Error("wanted 'context canceled', got:", err)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not exit properly from cancelling context")
	}
}

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
		{
			"overall status: success",
			http.StatusOK,
			OverallStatus{Status: StatusDegraded},
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
