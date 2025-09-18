package cs

// New returns a new cs object
func New() Config {
	return newCachedConfig()
}

// Get retrieves the value of type T associated with the specified key from the provided Config. Returns an error if retrieval fails.
func Get[T any](c Config, key string) (*T, error) {

	res := new(T)

	err := c.Read(key, res)

	return res, err
}

// MustGet retrieves the value of type T associated with the specified key from the provided Config. It panics on errors.
func MustGet[T any](c Config, key string) *T {

	res := new(T)
	c.MustRead(key, res)
	return res
}
