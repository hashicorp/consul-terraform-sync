// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package tmplfunc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServicesMeta_SetMetaMap(t *testing.T) {
	metaMap := map[string]map[string]string{
		"api": {
			"key": "value",
		},
	}

	t.Run("happy_path", func(t *testing.T) {
		m := &ServicesMeta{}

		err := m.SetMetaMap(metaMap)
		assert.NoError(t, err)

		assert.Equal(t, metaMap, m.metaMap)
		assert.Nil(t, m.meta)
	})

	t.Run("error_meta_already_set", func(t *testing.T) {
		m := &ServicesMeta{}

		// set meta first
		err := m.SetMeta(map[string]string{"key": "value"})
		assert.NoError(t, err)

		// confirm setting metamap errors
		err = m.SetMetaMap(metaMap)
		assert.Error(t, err)
	})
}

func TestServicesMeta_SetMeta(t *testing.T) {
	meta := map[string]string{"key": "value"}

	t.Run("happy_path", func(t *testing.T) {
		m := &ServicesMeta{}

		err := m.SetMeta(meta)
		assert.NoError(t, err)

		assert.Equal(t, meta, m.meta)
		assert.Nil(t, m.metaMap)
	})

	t.Run("error_meta_map_already_set", func(t *testing.T) {
		m := &ServicesMeta{}

		// set meta-map first
		err := m.SetMetaMap(map[string]map[string]string{
			"api": {
				"key": "value",
			},
		})
		assert.NoError(t, err)

		// confirm setting meta errors
		err = m.SetMeta(meta)
		assert.Error(t, err)
	})
}

func TestServicesMeta_Get(t *testing.T) {
	meta := map[string]string{
		"key": "value",
	}

	cases := []struct {
		name         string
		servicesMeta *ServicesMeta
	}{
		{
			"meta-map",
			&ServicesMeta{
				metaMap: map[string]map[string]string{
					"api": meta,
				},
			},
		},
		{
			"meta",
			&ServicesMeta{
				meta: meta,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.servicesMeta.Get("api")
			assert.Equal(t, meta, actual)
		})
	}
}
