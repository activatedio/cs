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

func (c *cachedConfig) GetDescriptions(key string) (map[string]any, error) {

	c.lock.RLock()

	if res, ok := c.descriptionsCache[key]; ok {
		defer c.lock.RUnlock()
		return res, nil
	}

	c.lock.RUnlock()

	c.lock.Lock()
	defer c.lock.Unlock()

	res, err := c.delegate.GetDescriptions(key)
	if err != nil {
		return nil, err
	}

	c.descriptionsCache[key] = res

	return res, nil
}

func (c *cachedConfig) MustGetDescriptions(key string) map[string]any {
	res, err := c.GetDescriptions(key)
	if err != nil {
		panic(err)
	}
	return res
}

func (c *cachedConfig) AddDefaultSource(src Source) {
	c.delegate.AddDefaultSource(src)
}

func (c *cachedConfig) SetValidatingHook(f func(in any) error) {
	c.delegate.SetValidatingHook(f)
}

func (c *cachedConfig) AddSource(src Source) {
	c.delegate.AddSource(src)
}

func (c *cachedConfig) AddLateBindingSource(src LateBindingSource) {
	c.delegate.AddLateBindingSource(src)
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
