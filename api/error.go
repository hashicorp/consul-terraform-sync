// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package api

// ErrorObject is the object to represent an error object from the API server
type ErrorObject struct {
	Message string `json:"message"`
}

// ErrorResponse is the object to represent an error response from the API server
type ErrorResponse struct {
	Error *ErrorObject `json:"error,omitempty"`
}

// NewErrorResponse creates a new API response for an error
func NewErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Error: &ErrorObject{
			Message: err.Error(),
		},
	}
}

// ErrorMessage returns the error message if there is an error.
func (resp ErrorResponse) ErrorMessage() (string, bool) {
	if resp.Error == nil {
		return "", false
	}

	return resp.Error.Message, true
}
