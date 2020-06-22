package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	jsonConfig = []byte(`{
	"log_level": "ERR",
	"inspect_mode": true
}`)

	hclConfig = []byte(`
	log_level = "ERR"
	inspect_mode = true
`)

	testConfig = Config{
		LogLevel:    String("ERR"),
		InspectMode: Bool(true),
	}
)

func TestDecodeConfig(t *testing.T) {
	testCases := []struct {
		name     string
		format   string
		content  []byte
		expected *Config
	}{
		{
			"hcl",
			"hcl",
			hclConfig,
			&testConfig,
		}, {
			"json",
			"json",
			jsonConfig,
			&testConfig,
		}, {
			"unsupported format",
			"txt",
			hclConfig,
			nil,
		}, {
			"hcl invalid",
			"hcl",
			[]byte(`log_level: "ERR"`),
			nil,
		}, {
			"hcl unexpected key",
			"hcl",
			[]byte(`key = "does_not_exist"`),
			nil,
		}, {
			"json invalid",
			"json",
			[]byte(`{"log_level" = "ERR"}`),
			nil,
		}, {
			"json unexpected key",
			"json",
			[]byte(`{"key": "does_not_exist"}`),
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := decodeConfig(tc.content, tc.format)
			if tc.expected == nil {
				assert.Error(t, err)
				return
			}

			require.NotNil(t, c)
			assert.Equal(t, *tc.expected, *c)
		})
	}
}

func TestFromPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected *Config
	}{
		{
			"load file",
			"testdata/simple.hcl",
			&Config{
				LogLevel:    String("ERR"),
				InspectMode: Bool(true),
			},
		}, {
			"load dir merge",
			"testdata/simple",
			&Config{
				LogLevel:    String("ERR"),
				InspectMode: Bool(true),
			},
		}, {
			"load dir override sorted by filename",
			"testdata/override",
			&Config{
				LogLevel:    String("DEBUG"),
				InspectMode: Bool(false),
			},
		}, {
			"file DNE",
			"testdata/dne.hcl",
			nil,
		}, {
			"dir DNE",
			"testdata/dne",
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := fromPath(tc.path)
			if tc.expected == nil {
				assert.Error(t, err)
				return
			}

			require.NotNil(t, c)
			assert.Equal(t, *tc.expected, *c)
		})
	}
}
