// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
		name        string
		checkReturn error
		statusCode  int
	}{
		{
			name:        "success",
			checkReturn: nil,
			statusCode:  http.StatusOK,
		},
		{
			name:        "failure checking health",
			checkReturn: errors.New("test error"),
			statusCode:  http.StatusInternalServerError,
		},
		{
			name:        "failure unhealthy system",
			checkReturn: &health.UnhealthySystemError{Err: errors.New("test error")},
			statusCode:  http.StatusServiceUnavailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checker := new(mockHealth.Checker)
			checker.On("Check").Return(tc.checkReturn)
			handler := NewHealthHandler(checker)

			path := healthPath
			req, err := http.NewRequest(http.MethodGet, path, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			handler.GetHealth(rr, req)

			var hcr oapigen.HealthCheckResponse
			err = json.NewDecoder(rr.Body).Decode(&hcr)
			assert.NoError(t, err)

			assert.Equal(t, rr.Code, tc.statusCode)
			if tc.checkReturn != nil {
				assert.NotNil(t, hcr.Error)
				assert.NotEmpty(t, hcr.Error.Message)
			}
		})
	}
}
