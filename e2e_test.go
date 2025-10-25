package cs_test

import (
	"os"
	"testing"

	"github.com/activatedio/cs"
	"github.com/activatedio/cs/sources"
	"github.com/activatedio/cs/sources/json"
	"github.com/activatedio/cs/sources/yaml"
	"github.com/stretchr/testify/assert"
)

type E2EConfig struct {
	Value1 string
	Value2 int
	Value3 bool
}

// TODO - test that the descriptions stay even once sources override them without descriptions

func TestMultipleSources(t *testing.T) {

	a := assert.New(t)

	unit := cs.New()

	unit.AddSource(yaml.FromPath("testdata/config.yaml", "prefixA"))
	unit.AddSource(json.FromPath("testdata/config.json", "prefixB"))
	unit.AddSource(yaml.FromPath("testdata/config.yaml", ""))
	unit.AddSource(json.FromPath("testdata/config.json", ""))
	unit.AddSource(sources.FromValue("prefixC", &E2EConfig{
		Value1: "a",
		Value2: 2,
		Value3: true,
	}))
	unit.AddLateBindingSource(sources.FromEnvironment("TEST"))

	res := map[string]any{}

	os.Setenv("TEST_DISPLAY_NAME", "Display Name Override")
	os.Setenv("TEST_PREFIX_A_DISPLAY_NAME", "Display Name Override - Prefix A")

	unit.MustRead("", &res)

	a.Equal(
		map[string]interface{}{
			"content": map[string]interface{}{
				"footer": "Some footer",
				"title":  "Some title"},
			"database": map[string]interface{}{
				"host": "dbhost",
				"user": "dbuser",
			},
			"devMode":     true,
			"displayName": "Display Name Override",
			"enabled":     true,
			"hostname":    "example.org",
			"numThreads":  float64(2),
			"prefixA": map[string]interface{}{
				"content": map[string]interface{}{
					"footer": "Some footer",
					"title":  "Some title",
				},
				"displayName":  "Display Name Override - Prefix A",
				"enabled":      true,
				"sleepSeconds": 60,
			},
			"prefixB": map[string]interface{}{
				"database": map[string]interface{}{
					"host": "dbhost",
					"user": "dbuser",
				},
				"devMode":    true,
				"hostname":   "example.org",
				"numThreads": float64(2),
			},
			"prefixC": map[string]interface{}{
				"value1": "a",
				"value2": 2,
				"value3": true,
			},
			"sleepSeconds": 60,
		},
		res)
}

type Dummy struct {
	Value1 string
	Value2 int
	Value3 bool
	Value4 string
}

func TestEnvironmentVariablesOnly(t *testing.T) {

	os.Setenv("TEST_DUMMY_VALUE1", "value1")
	os.Setenv("TEST_DUMMY_VALUE2", "1234")

	a := assert.New(t)

	unit := cs.New()
	unit.AddLateBindingSource(sources.FromEnvironment("TEST"))

	got := &Dummy{}

	unit.MustRead("dummy", got)

	expected := &Dummy{
		Value1: "value1",
		Value2: 1234,
	}

	a.Equal(expected, got)

}
