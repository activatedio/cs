package sources

import "github.com/activatedio/cs"

// NewSource returns the given value for a cs key
func NewSource(key string, val any) cs.Source {
	return func() (string, any, error) {
		return key, val, nil
	}
}
