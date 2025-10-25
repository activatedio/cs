package cs_test

import (
	"testing"

	"github.com/activatedio/cs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {

	// We simply test an error case for a ptr value
	c := cs.New()

	got, err := cs.Get[*string](c, "")

	require.EqualError(t, err, "type parameter must not be a pointer")
	assert.Nil(t, got)
}
