package version

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestTerraformConstraint(t *testing.T) {
	testCases := []struct {
		name      string
		version   string
		supported bool
	}{
		{
			"valid 0.13",
			"0.13.5",
			true,
		}, {
			"valid 0.14",
			"0.14.1",
			true,
		}, {
			"valid 0.15",
			"0.15.0",
			true,
		}, {
			"valid 1.0",
			"1.0.0",
			true,
		}, {
			"invalid lower bound",
			"0.12.12",
			false,
		}, {
			"invalid upper bound",
			"1.1.0",
			false,
		}, {
			"unsupported beta release",
			"0.14.0-beta1",
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := version.Must(version.NewSemver(tc.version))
			supported := TerraformConstraint.Check(v)
			assert.Equal(t, tc.supported, supported, tc.version)
		})
	}
}
