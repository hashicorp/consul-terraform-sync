package tftmpl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinStrings(t *testing.T) {
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
			actual := JoinStrings(".", tc.content...)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestHCLString(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			"empty",
			"",
			"null",
		}, {
			"string",
			"foobar",
			`"foobar"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := HCLString(tc.content)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestHCLStringList(t *testing.T) {
	testCases := []struct {
		name     string
		content  []string
		expected string
	}{
		{
			"nil",
			nil,
			"[]",
		}, {
			"empty",
			[]string{},
			"[]",
		}, {
			"list",
			[]string{"foo", "foobar", "foobarbaz"},
			`["foo", "foobar", "foobarbaz"]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := HCLStringList(tc.content)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestHCLStringMap(t *testing.T) {
	testCases := []struct {
		name     string
		content  map[string]string
		indent   int
		expected string
	}{
		{
			"nil",
			nil,
			0,
			"{}",
		}, {
			"empty",
			map[string]string{},
			0,
			"{}",
		}, {
			"map",
			map[string]string{"foo": "bar", "foobar": "foobarbaz"},
			0,
			`{ foo = "bar", foobar = "foobarbaz" }`,
		}, {
			"map sorts",
			map[string]string{"foobar": "foobarbaz", "foo": "bar"},
			0,
			`{ foo = "bar", foobar = "foobarbaz" }`,
		}, {
			"map with negative indent",
			map[string]string{"foo": "bar", "foobar": "foobarbaz"},
			-5,
			`{ foo = "bar", foobar = "foobarbaz" }`,
		}, {
			"map with indent",
			map[string]string{"foo": "bar", "foobar": "foobarbaz"},
			1,
			`{
  foo    = "bar"
  foobar = "foobarbaz"
}`,
		}, {
			"map with indents",
			map[string]string{"foo": "bar", "foobar": "foobarbaz"},
			3,
			`{
      foo    = "bar"
      foobar = "foobarbaz"
    }`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := HCLStringMap(tc.content, tc.indent)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
