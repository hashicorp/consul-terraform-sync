package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/managers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HealthHandler_GetHealth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		setup    func(m *mocks.Health)
		validate func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name: "success",
			setup: func(m *mocks.Health) {
				m.On("Check").Return(nil)
			},
			validate: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)

				var hcr oapigen.GoodHealthCheckResponse
				err := json.NewDecoder(resp.Body).Decode(&hcr)
				assert.NoError(t, err)

				ts, err := time.Parse(time.RFC3339, fmt.Sprint(hcr.Timestamp))
				assert.NoError(t, err)
				assert.WithinDuration(t, time.Now().UTC(), ts, time.Second)
			},
		},
		{
			name: "failure",
			setup: func(m *mocks.Health) {
				m.On("Check").Return(errors.New("test error"))
			},
			validate: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)

				var hcr oapigen.BadHealthCheckResponse
				err := json.NewDecoder(resp.Body).Decode(&hcr)
				assert.NoError(t, err)

				ts, err := time.Parse(time.RFC3339, fmt.Sprint(hcr.Timestamp))
				assert.NoError(t, err)
				assert.WithinDuration(t, time.Now().UTC(), ts, time.Second)
				assert.Equal(t, "test error", hcr.Error.Message)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hm := new(mocks.Health)
			tc.setup(hm)
			handler := NewHealthHandler(hm)

			path := "/v1/health"
			req, err := http.NewRequest(http.MethodGet, path, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.GetHealth(resp, req)
			tc.validate(t, resp)
		})
	}
}
