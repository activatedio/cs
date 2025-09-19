// Package testdata contains test data
package testdata

import _ "embed"

//go:embed config.yaml
var Config []byte

//go:embed invalid.yaml
var Invalid []byte
