// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
)

type contextReqKeyType struct{}

var reqContextKey = contextReqKeyType{}

// requestIDWithContext inserts a requestID into the context and is retrievable
// with FromContext.
func requestIDWithContext(ctx context.Context, requestID string) context.Context {
	// While we could call logger.With even with zero args, we have this
	// check to avoid unnecessary allocations around creating a copy of a
	// logger.

	reqID := uuid.MustParse(requestID)
	return context.WithValue(ctx, reqContextKey, reqID)
}

// requestIDFromContext retrieves a requestID from the context if one exists, and returns
// and empty string otherwise.
func requestIDFromContext(ctx context.Context) oapigen.RequestID {
	requestID, _ := ctx.Value(reqContextKey).(oapigen.RequestID)

	return requestID
}
