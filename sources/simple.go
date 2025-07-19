package sources

import "github.com/activatedio/config"

// NewSource returns the given value for a config key
func NewSource(key string, val any) config.Source {
	return func() (string, any, error) {
		return key, val, nil
	}
}
