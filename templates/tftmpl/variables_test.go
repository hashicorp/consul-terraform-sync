package tftmpl

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRootVariables(t *testing.T) {
	goldenFile := filepath.Join("testdata", "variables.tf.golden")
	input := RootModuleInputData{
		Providers: []map[string]interface{}{{
			"testProvider": map[string]interface{}{
				"alias": "tp",
				"attr":  "value",
				"count": 10,
			},
		}},
	}
	input.Init()
	b := new(bytes.Buffer)
	err := NewVariablesTF(b, &input)
	require.NoError(t, err)
	checkGoldenFile(t, goldenFile, b.Bytes())
}
