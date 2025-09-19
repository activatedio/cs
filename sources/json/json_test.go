package json_test

import (
	"bytes"
	"testing"

	"github.com/activatedio/cs/sources/json"
	"github.com/activatedio/cs/sources/json/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromPath(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	key, data, err := json.FromPath("./testdata/config.json", "test")()

	r.NoError(err)
	a.Equal("test", key)
	a.Equal(map[string]any{
		"a": "1",
		"b": "2",
	}, data)

	key, data, err = json.FromPath("./testdata/invalid.json", "test")()

	a.Empty(key)
	a.Nil(data)
	r.EqualError(err, "invalid character 'i' looking for beginning of object key string")

	key, data, err = json.FromPath("./testdata/nopath.json", "test")()

	a.Empty(key)
	a.Nil(data)
	r.EqualError(err, "open ./testdata/nopath.json: no such file or directory")

}
func TestFromReader(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	key, data, err := json.FromReader(bytes.NewReader(testdata.Config), "test")()

	r.NoError(err)
	a.Equal("test", key)
	a.Equal(map[string]any{
		"a": "1",
		"b": "2",
	}, data)

	key, data, err = json.FromReader(bytes.NewReader(testdata.Invalid), "test")()

	a.Empty(key)
	a.Nil(data)
	r.EqualError(err, "invalid character 'i' looking for beginning of object key string")

}
