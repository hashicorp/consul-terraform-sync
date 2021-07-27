package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected error
	}{
		{
			"nil",
			nil,
			nil,
		}, {
			"no change",
			errors.New("no change"),
			errors.New("no change"),
		}, {
			"unchanged decode error",
			errors.New(`'file.hcl' has invalid keys: consul.unsupported`),
			errors.New(`'file.hcl' has invalid keys: consul.unsupported`),
		}, {
			"unexpected type",
			errors.New(`* 'consul' expected a map, got 'int'`),
			errors.New(`* 'consul' expected a map, got 'int'`),
		}, {
			"unsupported repeated blocks",
			errors.New(`1 error(s) decoding:

* 'driver' expected a map, got 'slice'
`),
			errors.New("only one 'driver' block can be configured"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := decodeError(tc.err)
			assert.Equal(t, tc.expected, err)
		})
	}
}
