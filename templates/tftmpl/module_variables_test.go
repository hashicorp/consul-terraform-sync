// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftmpl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVariables(t *testing.T) {
	testCases := []struct {
		name    string
		content []byte
		err     bool
	}{
		{
			"valid types",
			[]byte(`
b = true
key = "some_key"
num = 10
obj = {
  argStr = "value"
  argNum = 10
  argList = ["l", "i", "s", "t"]
  argMap = {}
}
l = [1, 2, 3]
tup = ["abc", 123, true]`),
			false,
		}, {
			"unsupported type",
			[]byte(`b = true + 1`),
			true,
		}, {
			"invalid syntax",
			[]byte(`key = "missing closing quote`),
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vars, err := ParseModuleVariables(tc.content, "filename")
			if tc.err {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, vars, 6)
		})
	}
}

func TestParseVariablesFromMap(t *testing.T) {
	testCases := []struct {
		name    string
		content map[string]string
		err     bool
	}{
		{
			name: "valid types",
			content: map[string]string{
				"b":   "true",
				"key": `"some_key"`,
				"num": "10",
				"l":   "[1,2,3]",
				"tup": `["abc", 123, true]`,
				"obj": `{
  argStr = "value"
  argNum = 10
  argList = ["l", "i", "s", "t"]
  argMap = {}
}`,
			},
			err: false,
		}, {
			"unsupported type",
			map[string]string{
				"b": "true + 1",
			},
			true,
		}, {
			"invalid syntax",
			map[string]string{
				"key": `"missing closing quote`,
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vars, err := ParseModuleVariablesFromMap(tc.content)
			if tc.err {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, vars, 6)
		})
	}
}
