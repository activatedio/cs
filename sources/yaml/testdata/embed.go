// Package testdata contains test data
package testdata

import _ "embed"

//go:embed config.yaml
var Config []byte

//go:embed invalid.yaml
var Invalid []byte

//go:embed config_env.yaml
var ConfigEnv []byte
