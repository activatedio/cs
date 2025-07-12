package yaml

import (
	"github.com/activatedio/config"
	"gopkg.in/yaml.v3"
	"os"
)

func NewSourceFromPath(path, keyPrefix string) config.Source {
	return func() (string, any, error) {

		res := map[string]any{}

		f, err := os.Open(path)

		if err != nil {
			return "", nil, err
		}

		defer f.Close()

		err = yaml.NewDecoder(f).Decode(&res)

		if err != nil {
			return "", nil, err
		}

		return keyPrefix, res, nil
	}
}
