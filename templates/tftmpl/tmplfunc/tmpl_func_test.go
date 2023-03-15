// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tmplfunc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinStringsFunc(t *testing.T) {
	testCases := []struct {
		name     string
		content  []string
		expected string
	}{
		{
			"empty",
			[]string{},
			"",
		}, {
			"string",
			[]string{"foobar"},
			"foobar",
		}, {
			"multiple strings",
			[]string{"foo", "bar", "baz"},
			"foo.bar.baz",
		}, {
			"empty string ignored",
			[]string{"foo", "", "baz"},
			"foo.baz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := joinStringsFunc(".", tc.content...)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
