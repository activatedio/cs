package cs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/spf13/cast"
)

type cs struct {
	sources            []Source
	lateBindingSources []LateBindingSource
	dirty              bool
	root               map[string]reflect.Value
	lock               sync.RWMutex
}

func (c *cs) AddSource(src Source) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.sources = append(c.sources, src)
	c.dirty = true
}

func (c *cs) AddLateBindingSource(src LateBindingSource) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lateBindingSources = append(c.lateBindingSources, src)
	c.dirty = true
}

func (c *cs) loadData() error {

	c.lock.Lock()
	defer c.lock.Unlock()

	c.root = make(map[string]reflect.Value)

	for _, src := range c.sources {
		key, v, err := src()
		if err != nil {
			return err
		}
		var val reflect.Value
		val, err = c.toValue(v)
		if err != nil {
			return err
		}
		var tmp map[string]reflect.Value
		tmp, err = c.toValueMap(key, val)
		if err != nil {
			return err
		}
		// We ignore return as maps are never replaced
		_, err = c.replaceOrMergeValues(reflect.ValueOf(c.root), reflect.ValueOf(tmp))
		if err != nil {
			return err
		}
	}

	c.dirty = false

	return nil
}

func (c *cs) toValueMap(key string, v reflect.Value) (map[string]reflect.Value, error) {

	if key == "" {
		if val, ok := v.Interface().(map[string]reflect.Value); ok {
			return val, nil
		}
		return nil, errors.New("invalid root type")
	}
	// Build out a map structure
	val := map[string]reflect.Value{}
	parts := strings.SplitN(key, ".", 2)
	thisKey := parts[0]
	if len(parts) == 1 {
		val[thisKey] = v
	} else {
		rest := parts[1]
		tmp, err := c.toValueMap(rest, v)
		if err != nil {
			return nil, err
		}
		val[thisKey] = reflect.ValueOf(tmp)
	}
	return val, nil
}

func (c *cs) toValue(v any) (reflect.Value, error) {
	typ := reflect.TypeOf(v)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		// We assume we can interface this
		v = reflect.Indirect(reflect.ValueOf(v)).Interface()
	}

	switch typ.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		return reflect.ValueOf(v), nil
	case reflect.Map:
		return c.toValueFromMap(v)
	case reflect.Struct:
		return c.toValueFromStruct(v)
	default:
		return reflect.ValueOf(nil), fmt.Errorf("unsupported kind %s", typ.Kind().String())
	}
}

func (c *cs) toValueFromMap(v any) (reflect.Value, error) {
	// We assume this is a struct and convert this to a map of values
	res := map[string]reflect.Value{}
	if val, ok := v.(map[string]any); ok {
		for k, _v := range val {
			fv, err := c.toValue(_v)
			if err != nil {
				return reflect.Value{}, err
			}
			res[k] = fv
		}
	} else {
		return reflect.ValueOf(nil), errors.New("map must be of type map[string]any")
	}

	return reflect.ValueOf(res), nil
}

func (c *cs) toValueFromStruct(v any) (reflect.Value, error) {
	// We assume this is a struct and convert this to a map of values
	res := map[string]reflect.Value{}
	val := reflect.ValueOf(v)

	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		if f.CanInterface() {
			fv, err := c.toValue(f.Interface())
			if err != nil {
				return reflect.Value{}, err
			}
			res[toLowerCamel(val.Type().Field(i).Name)] = fv
		}
	}

	return reflect.ValueOf(res), nil
}

func (c *cs) fromValue(fullKey string, val reflect.Value, into any) error {
	dest := reflect.ValueOf(into)
	if dest.Kind() == reflect.Ptr {
		dest = dest.Elem()
	}
	return c.populateValue(fullKey, dest, val)
}

func (c *cs) populateValue(fullKey string, dest reflect.Value, val reflect.Value) error {
	switch dest.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:

		for _, src := range c.lateBindingSources {
			lbVal, err := src(fullKey)
			if err != nil {
				return err
			}
			if lbVal != nil {
				val = reflect.ValueOf(lbVal)
			}
		}

		// This skips the set in case of a zero value
		if val.IsValid() {
			// Need some type conversions, especially given some of the late binding sources will be strings from
			// environment variables
			err := c.castAndSet(dest, val)
			if err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		// We need to be able to write to the struct
		return c.populateMap(fullKey, dest, val)
	case reflect.Struct:
		// We need to be able to write to the struct
		return c.populateStruct(fullKey, dest, val)
	default:
		return fmt.Errorf("unsupported destination kind %s", dest.Kind().String())
	}
}

func (c *cs) castAndSet(dest, src reflect.Value) error { //nolint:gocyclo // switch statement ok for readability

	if dest.Type() == src.Type() {
		dest.Set(src)
		return nil
	}

	var val any

	switch dest.Kind() {
	case reflect.String:
		val = cast.ToString(src.Interface())
	case reflect.Int:
		val = cast.ToInt(src.Interface())
	case reflect.Int8:
		val = cast.ToInt8(src.Interface())
	case reflect.Int16:
		val = cast.ToInt16(src.Interface())
	case reflect.Int32:
		val = cast.ToInt32(src.Interface())
	case reflect.Int64:
		val = cast.ToInt64(src.Interface())
	case reflect.Uint:
		val = cast.ToUint(src.Interface())
	case reflect.Uint8:
		val = cast.ToUint8(src.Interface())
	case reflect.Uint16:
		val = cast.ToUint16(src.Interface())
	case reflect.Uint32:
		val = cast.ToUint32(src.Interface())
	case reflect.Uint64:
		val = cast.ToUint64(src.Interface())
	case reflect.Float32:
		val = cast.ToFloat32(src.Interface())
	case reflect.Float64:
		val = cast.ToFloat64(src.Interface())
	case reflect.Bool:
		val = cast.ToBool(src.Interface())
	default:
		return fmt.Errorf("unsupported type %s", dest.Type().String())
	}

	dest.Set(reflect.ValueOf(val))

	return nil
}

var typeMapStringReflectValue = reflect.TypeFor[map[string]reflect.Value]()
var typeMapStringAny = reflect.TypeFor[map[string]any]()

func (c *cs) populateMap(fullKey string, dest reflect.Value, val reflect.Value) error {

	// type must be map[string]reflect.Value
	if val.Kind() != reflect.Map {
		// Value is not a map, can't do anything
		return nil
	}

	vInt := val.Interface()
	dInt := dest.Interface()

	if _, ok := vInt.(map[string]reflect.Value); !ok {
		return errors.New("invalid internal map type. must be map[string]reflect.Value")
	}

	if _, ok := dInt.(map[string]any); !ok {
		return errors.New("invalid destination map type. must be map[string]any")
	}

	for _, key := range val.MapKeys() {
		exist := dest.MapIndex(key)
		var _fullKey string
		if fullKey == "" {
			_fullKey = toLowerCamel(key.String())
		} else {
			_fullKey = fmt.Sprintf("%s.%s", fullKey, toLowerCamel(key.String()))
		}
		_val := val.MapIndex(key)
		if exist.IsValid() {
			err := c.populateValue(_fullKey, exist, _val)
			if err != nil {
				return err
			}
		} else {
			tmp := _val.Interface().(reflect.Value)
			_type := tmp.Type()
			var _dest reflect.Value
			if _type == typeMapStringReflectValue {
				_dest = reflect.MakeMap(typeMapStringAny)
			} else {
				_dest = reflect.New(_type).Elem()
			}
			err := c.populateValue(_fullKey, _dest, tmp)
			if err != nil {
				return err
			}
			dest.SetMapIndex(key, _dest)
		}
	}
	return nil
}

func (c *cs) populateStruct(fullKey string, dest reflect.Value, val reflect.Value) error {

	// type must be map[string]reflect.Value
	if val.Kind() != reflect.Map {
		// Value is not a map, can't do anything
		return nil
	}

	// TODO - consider keys which are not present in the map

	if valMap, valMapOk := val.Interface().(map[string]reflect.Value); valMapOk {

		for i := 0; i < dest.NumField(); i++ {
			f := dest.Field(i)
			name := toLowerCamel(dest.Type().Field(i).Name)
			v := valMap[name]
			err := c.populateValue(fmt.Sprintf("%s.%s", fullKey, name), f, v)
			if err != nil {
				return err
			}
		}
	} else {
		// Can't do anything, return nil
		return nil
	}

	return nil
}

func (c *cs) replaceOrMergeValues(existing reflect.Value, value reflect.Value) (reflect.Value, error) {

	switch existing.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		if value.Kind() == reflect.Map {
			return reflect.Value{}, fmt.Errorf("cannot overrwrite type %s with a map", existing.Kind().String())
		}
		return value, nil
	case reflect.Map:
		if value.Kind() != reflect.Map {
			return reflect.Value{}, fmt.Errorf("invalid value for map target %s", value.Kind().String())
		}
		// New must also be the same type
		if eMap, eOk := existing.Interface().(map[string]reflect.Value); eOk {
			if nMap, nOk := value.Interface().(map[string]reflect.Value); nOk {
				for k, v := range nMap {
					if el, elOk := eMap[k]; elOk {
						// map contains value, we merge
						var err error
						v, err = c.replaceOrMergeValues(el, v)
						if err != nil {
							return reflect.Value{}, err
						}
					}
					eMap[k] = v
				}
			} else {
				return reflect.Value{}, errors.New("new is unexpectedly not a map[string]reflect.Value")
			}
			return existing, nil
		}
		return reflect.Value{}, errors.New("destination is unexpectedly not a map[string]reflect.Value")
	default:
		return reflect.Value{}, fmt.Errorf("unsupported existing kind %s", existing.Kind().String())
	}
}

func (c *cs) withCleanData(callback func() error) error {
	c.lock.RLock()

	if c.dirty {
		c.lock.RUnlock()

		// Build data from sources
		err := c.loadData()

		if err != nil {
			return err
		}

		// Re-establish the read lock
		c.lock.RLock()
	}

	defer c.lock.RUnlock()

	return callback()

}

func (c *cs) read(fullKey, key string, data map[string]reflect.Value, into any) error {
	parts := strings.SplitN(key, ".", 2)
	thisKey := parts[0]
	if thisKey == "" {
		// Special case for root of the cs
		return c.fromValue("", reflect.ValueOf(c.root), into)
	}
	if tmp, ok := data[thisKey]; ok {
		if len(parts) == 1 {
			return c.fromValue(fullKey, tmp, into)
		} else if data, ok = tmp.Interface().(map[string]reflect.Value); ok {
			return c.read(fullKey, parts[1], data, into)
		}
		return fmt.Errorf("invalid type for key %s", thisKey)
	}
	// We still populate the value in the case it is a struct and we can lookup keys based on fields
	return c.fromValue(fullKey, reflect.New(typeMapStringReflectValue).Elem(), into)
}

func (c *cs) Read(key string, into any) error {
	if reflect.ValueOf(into).Kind() != reflect.Ptr {
		return errors.New("into must be a pointer")
	}
	return c.withCleanData(func() error {
		return c.read(key, key, c.root, into)
	})
}

func (c *cs) MustRead(key string, into any) {
	if err := c.Read(key, into); err != nil {
		panic(err)
	}
}

func newConfig() Config {
	return &cs{
		root: map[string]reflect.Value{},
	}
}
