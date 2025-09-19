package yaml_test

import (
	"bytes"
	"testing"

	"github.com/activatedio/cs/sources/yaml"
	"github.com/activatedio/cs/sources/yaml/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromPath(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	key, data, err := yaml.FromPath("./testdata/config.yaml", "test")()

	r.NoError(err)
	a.Equal("test", key)
	a.Equal(map[string]any{
		"a": "1",
		"b": "2",
	}, data)

	key, data, err = yaml.FromPath("./testdata/invalid.yaml", "test")()

	a.Empty(key)
	a.Nil(data)
	r.EqualError(err, "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `not yaml` into map[string]interface {}")

	key, data, err = yaml.FromPath("./testdata/nopath.yaml", "test")()

	a.Empty(key)
	a.Nil(data)
	r.EqualError(err, "open ./testdata/nopath.yaml: no such file or directory")

}

func TestFromReader(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	key, data, err := yaml.FromReader(bytes.NewReader(testdata.Config), "test")()

	r.NoError(err)
	a.Equal("test", key)
	a.Equal(map[string]any{
		"a": "1",
		"b": "2",
	}, data)

	key, data, err = yaml.FromReader(bytes.NewReader(testdata.Invalid), "test")()

	a.Empty(key)
	a.Nil(data)
	r.EqualError(err, "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `not yaml` into map[string]interface {}")

}
