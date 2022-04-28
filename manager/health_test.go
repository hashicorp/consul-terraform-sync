package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BasicHealth_Check(t *testing.T) {
	t.Parallel()

	h := &BasicHealth{}
	assert.NoError(t, h.Check())
}
