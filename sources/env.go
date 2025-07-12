package sources

import (
	"fmt"
	"github.com/activatedio/config"
	"os"
	"regexp"
	"strings"
)

// From https://stackoverflow.com/a/56616250
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func NewEnvLateBindingSource(envPrefix string) config.LateBindingSource {
	return func(key string) (any, error) {

		snake := matchFirstCap.ReplaceAllString(key, "${1}_${2}")
		snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}") // Translate the key to an env formatted name
		snake = strings.ReplaceAll(snake, ".", "_")
		snake = strings.ToUpper(snake)

		if envPrefix != "" {
			snake = fmt.Sprintf("%s_%s", envPrefix, snake)
		}

		val := os.Getenv(snake)
		if val == "" {
			return nil, nil
		} else {
			return val, nil
		}
	}
}
