package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/health"
	mockHealth "github.com/hashicorp/consul-terraform-sync/mocks/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HealthHandler_GetHealth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		setup    func(m *mockHealth.Checker)
		validate func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name: "success",
			setup: func(m *mockHealth.Checker) {
				m.On("Check").Return(nil)
			},
			validate: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)

				var hcr oapigen.GoodHealthCheckResponse
				err := json.NewDecoder(resp.Body).Decode(&hcr)
				assert.NoError(t, err)

				assert.NoError(t, err)
			},
		},
		{
			name: "failure checking health",
			setup: func(m *mockHealth.Checker) {
				m.On("Check").Return(errors.New("test error"))
			},
			validate: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)

				var hcr oapigen.BadHealthCheckResponse
				err := json.NewDecoder(resp.Body).Decode(&hcr)
				assert.NoError(t, err)

				assert.NoError(t, err)
				assert.Equal(t, "test error", hcr.Error.Message)
			},
		},
		{
			name: "failure unhealthy system",
			setup: func(m *mockHealth.Checker) {
				m.On("Check").Return(&health.UnhealthySystemError{Err: errors.New("test error")})
			},
			validate: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

				var hcr oapigen.BadHealthCheckResponse
				err := json.NewDecoder(resp.Body).Decode(&hcr)
				assert.NoError(t, err)

				assert.NoError(t, err)
				assert.Equal(t, "CTS is not healthy: test error", hcr.Error.Message)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checker := new(mockHealth.Checker)
			tc.setup(checker)
			handler := NewHealthHandler(checker)

			path := "/v1/health"
			req, err := http.NewRequest(http.MethodGet, path, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.GetHealth(resp, req)
			tc.validate(t, resp)
		})
	}
}
