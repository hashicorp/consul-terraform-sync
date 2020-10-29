package driver

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestIsTFCompatible(t *testing.T) {
	cases := []struct {
		name       string
		version    *version.Version
		config     *config.TerraformConfig
		compatible bool
	}{
		{
			"valid",
			version.Must(version.NewSemver("0.13.2")),
			&config.TerraformConfig{
				Backend: make(map[string]interface{}),
			},
			true,
		}, {
			"pg backend compatible",
			version.Must(version.NewSemver("0.14.0")),
			&config.TerraformConfig{
				Backend: map[string]interface{}{"pg": nil},
			},
			true,
		}, {
			"pg backend incompatible",
			version.Must(version.NewSemver("0.13.5")),
			&config.TerraformConfig{
				Backend: map[string]interface{}{"pg": nil},
			},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := isTFCompatible(tc.config, tc.version)
			if tc.compatible {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
