package sources

import "github.com/activatedio/config"

func NewSource(key string, val any) config.Source {
	return func() (string, any, error) {
		return key, val, nil
	}
}
