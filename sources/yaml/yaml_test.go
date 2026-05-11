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

func TestFromPath_ExpandEnv(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	t.Setenv("TEST_EXPAND_A", "alpha")
	t.Setenv("TEST_EXPAND_B", "beta")

	// Without ExpandEnv, ${VAR} references are preserved as literal strings.
	key, data, err := yaml.FromPath("./testdata/config_env.yaml", "test")()
	r.NoError(err)
	a.Equal("test", key)
	a.Equal(map[string]any{
		"a": "${TEST_EXPAND_A}",
		"b": "static-${TEST_EXPAND_B}-suffix",
		"c": map[string]any{
			"d": "${TEST_EXPAND_A}",
			"e": []any{"${TEST_EXPAND_B}", "plain"},
		},
		"f": "${TEST_EXPAND_MISSING}",
	}, data)

	// With ExpandEnv, every string leaf is run through os.ExpandEnv,
	// including values nested in maps and slices. Unset vars expand to "".
	key, data, err = yaml.FromPath("./testdata/config_env.yaml", "test", yaml.ExpandEnv())()
	r.NoError(err)
	a.Equal("test", key)
	a.Equal(map[string]any{
		"a": "alpha",
		"b": "static-beta-suffix",
		"c": map[string]any{
			"d": "alpha",
			"e": []any{"beta", "plain"},
		},
		"f": "",
	}, data)
}

func TestFromReader_ExpandEnv(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	t.Setenv("TEST_EXPAND_A", "alpha")
	t.Setenv("TEST_EXPAND_B", "beta")

	key, data, err := yaml.FromReader(bytes.NewReader(testdata.ConfigEnv), "test", yaml.ExpandEnv())()
	r.NoError(err)
	a.Equal("test", key)
	a.Equal(map[string]any{
		"a": "alpha",
		"b": "static-beta-suffix",
		"c": map[string]any{
			"d": "alpha",
			"e": []any{"beta", "plain"},
		},
		"f": "",
	}, data)
}
