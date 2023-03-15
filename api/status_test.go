// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusHandlerDefault_GetClusterStatus(t *testing.T) {
	t.Parallel()

	handler := &StatusHandlerDefault{}

	resp, reqID := runTestGetClusterStatus(t, handler, http.StatusMethodNotAllowed)

	// Check response
	decoder := json.NewDecoder(resp.Body)
	var actual oapigen.ErrorResponse
	err := decoder.Decode(&actual)
	require.NoError(t, err)

	expected := generateErrorResponse(reqID.String(), haNotAvailableError)
	assert.Equal(t, expected, actual)
}

// runTestGetClusterStatus takes a handler and expected status and performs the
// Get operation for the `/v1/status/cluster` path. It returns the response
// and the request ID
func runTestGetClusterStatus(t *testing.T, handler StatusHandler, expectedStatus int) (*httptest.ResponseRecorder, uuid.UUID) {
	path := "/v1/status/cluster"
	reqID := uuid.New()

	// Create the request and recorder
	ctx := requestIDWithContext(context.Background(), reqID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	require.NoError(t, err)
	resp := httptest.NewRecorder()

	// Hit the /v1/status/cluster endpoint and check response code
	handler.GetClusterStatus(resp, req)
	require.Equal(t, expectedStatus, resp.Code)

	return resp, reqID
}
