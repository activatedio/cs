// Package testdata contains test data
package testdata

import _ "embed"

//go:embed config.json
var Config []byte

//go:embed invalid.json
var Invalid []byte

//go:embed config_env.json
var ConfigEnv []byte
