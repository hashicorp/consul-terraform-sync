package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

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
	defer ctx.Done()
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
			require.Equal(t, tc.statusCode, resp.StatusCode)
		})
	}
}
