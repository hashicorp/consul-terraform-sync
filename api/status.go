// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"errors"
	"net/http"
)

const (
	haNotAvailableError = "this endpoint is only available with high availability configured"
)

// StatusHandler is an interface for handling all status related endpoints
type StatusHandler interface {
	GetClusterStatus(w http.ResponseWriter, r *http.Request)
}

// StatusHandlerDefault is the default status handler
type StatusHandlerDefault struct{}

// GetClusterStatus returns an error message that this endpoint is not supported
// by non HA configured CTS instances
func (StatusHandlerDefault) GetClusterStatus(w http.ResponseWriter, r *http.Request) {
	endpointNotAvailableSender(haNotAvailableError, w, r)
}

func endpointNotAvailableSender(errMessage string, w http.ResponseWriter, r *http.Request) {
	err := errors.New(errMessage)
	sendError(w, r, http.StatusMethodNotAllowed, err)
}
