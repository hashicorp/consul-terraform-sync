// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package health

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BasicHealth_Check(t *testing.T) {
	t.Parallel()

	h := &BasicChecker{}
	assert.NoError(t, h.Check())
}

func TestUnhealthySystemError_Error(t *testing.T) {
	err := UnhealthySystemError{Err: errors.New("some error")}
	var nonEnterpriseConsulError *UnhealthySystemError

	assert.True(t, errors.As(&err, &nonEnterpriseConsulError))
	assert.Equal(t, "CTS is not healthy: some error", err.Error())
}

func TestUnhealthySystemError_Unwrap(t *testing.T) {
	var terr *testError

	var otherErr testError
	err := UnhealthySystemError{Err: &otherErr}

	// Assert that the wrapped error is still detectable
	// errors.As is the preferred way to call the underlying Unwrap
	assert.True(t, errors.As(&err, &terr))
}

type testError struct {
}

// Error returns an error string
func (e *testError) Error() string {
	return "this is a test error"
}
