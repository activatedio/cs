package cs

import (
	"reflect"
	"sync"
)

type cacheKey struct {
	Key string
	Typ reflect.Type
}

type cachedConfig struct {
	delegate          Config
	cache             map[cacheKey]reflect.Value
	descriptionsCache map[string]map[string]any
	lock              sync.RWMutex
}

func (c *cachedConfig) Dump(opts ...DumpOption) error {
	return c.delegate.Dump(opts...)
}

func (c *cachedConfig) AddDefaultSource(src Source) {
	c.delegate.AddDefaultSource(src)
	c.invalidate()
}

func (c *cachedConfig) SetValidatingHook(f func(in any) error) {
	c.delegate.SetValidatingHook(f)
}

func (c *cachedConfig) AddSource(src Source) {
	c.delegate.AddSource(src)
	c.invalidate()
}

func (c *cachedConfig) AddLockedSource(src Source) {
	c.delegate.AddLockedSource(src)
	c.invalidate()
}

func (c *cachedConfig) AddLateBindingSource(src LateBindingSource) {
	c.delegate.AddLateBindingSource(src)
	c.invalidate()
}

// invalidate clears the read caches after a source changes, so a value read
// before the source was added is recomputed on the next read rather than
// served stale. Without this, a locked source added after a prior read of the
// same key would silently no-op.
func (c *cachedConfig) invalidate() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache = map[cacheKey]reflect.Value{}
	c.descriptionsCache = map[string]map[string]any{}
}

func (c *cachedConfig) Read(key string, into any) error {
	c.lock.RLock()

	typ := reflect.TypeOf(into)
	val := reflect.ValueOf(into)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	if res, ok := c.cache[cacheKey{
		Key: key,
		Typ: typ,
	}]; ok {
		defer c.lock.RUnlock()
		val.Set(res)
		return nil
	}

	c.lock.RUnlock()

	c.lock.Lock()
	defer c.lock.Unlock()

	err := c.delegate.Read(key, into)
	if err != nil {
		return err
	}

	val = reflect.ValueOf(into)
	typ = reflect.TypeOf(into)
	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	c.cache[cacheKey{Key: key, Typ: typ}] = val

	return nil
}

func (c *cachedConfig) MustRead(key string, into any) {
	if err := c.Read(key, into); err != nil {
		panic(err)
	}
}

func newCachedConfig() Config {
	return &cachedConfig{
		delegate:          newConfig(),
		cache:             map[cacheKey]reflect.Value{},
		descriptionsCache: map[string]map[string]any{},
	}
}
