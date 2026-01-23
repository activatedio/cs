package cs

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cast"
)

var (
	typeEntry         = reflect.TypeFor[Entry]()
	typeMapStringNode = reflect.TypeFor[map[string]node]()
	typeArrayNode     = reflect.TypeFor[[]node]()
	typeMapStringAny  = reflect.TypeFor[map[string]any]()
)

type metadata struct {
	description string
	optional    bool
}

type node struct {
	meta metadata
	// for arrays which need a type
	sliceType reflect.Type
	nilType   reflect.Type
	value     reflect.Value
}

func (n node) interfaceOrNil() any {

	if n.value.IsValid() && n.value.CanInterface() {
		return n.value.Interface()
	}
	return nil
}

func (n node) getEffectiveType() reflect.Type {
	if n.value.Kind() == reflect.Invalid {
		return n.nilType
	}
	return n.value.Type()
}

type cs struct {
	sources            []Source
	lateBindingSources []LateBindingSource
	dirty              bool
	root               map[string]node
	lock               sync.RWMutex
	validatingHook     func(in any) error
}

func (c *cs) SetValidatingHook(f func(in any) error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.validatingHook = f
}

func (c *cs) AddDefaultSource(src Source) {

	c.lock.Lock()
	defer c.lock.Unlock()

	// prepend
	c.sources = append([]Source{src}, c.sources...)
	c.dirty = true
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

	c.root = make(map[string]node)

	for _, src := range c.sources {
		key, v, err := src()
		if err != nil {
			return err
		}
		var n node
		n, err = c.toNode(v, metadata{})
		if err != nil {
			return err
		}
		var tmp map[string]node
		tmp, err = c.toNodeMap(key, n)
		if err != nil {
			return err
		}
		// We ignore return as maps are never replaced
		_, err = c.replaceOrMerge(node{value: reflect.ValueOf(c.root)}, node{value: reflect.ValueOf(tmp)})
		if err != nil {
			return err
		}
	}

	c.dirty = false

	return nil
}

func (c *cs) toNodeMap(key string, n node) (map[string]node, error) {

	if key == "" {
		if val, ok := n.value.Interface().(map[string]node); ok {
			return val, nil
		}
		return nil, errors.New("invalid root type")
	}
	// Build out a map structure
	val := map[string]node{}
	parts := strings.SplitN(key, ".", 2)
	thisKey := parts[0]
	if len(parts) == 1 {
		val[thisKey] = n
	} else {
		rest := parts[1]
		tmp, err := c.toNodeMap(rest, n)
		if err != nil {
			return nil, err
		}
		val[thisKey] = node{value: reflect.ValueOf(tmp)}
	}
	return val, nil
}

func (c *cs) toNode(v any, meta metadata) (node, error) {

	typ := reflect.TypeOf(v)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		meta.optional = true
		// We assume we can interface this
		i := reflect.Indirect(reflect.ValueOf(v))
		if i.IsValid() {
			v = i.Interface()
		} else {
			return node{
				meta: metadata{
					optional:    true,
					description: meta.description,
				},
				nilType: typ,
			}, nil
		}
	}

	switch typ.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		return node{
			meta:  meta,
			value: reflect.ValueOf(v),
		}, nil
	case reflect.Map:
		return c.toNodeFromMap(v, meta)
	case reflect.Struct:
		return c.toNodeFromStruct(v, meta)
	case reflect.Slice:
		return c.toNodeFromSlice(v, meta)
	default:
		return node{}, fmt.Errorf("unsupported kind %s", typ.Kind().String())
	}
}

func (c *cs) toNodeFromMap(v any, meta metadata) (node, error) {
	// We assume this is a struct and convert this to a map of values
	res := map[string]node{}
	if val, ok := v.(map[string]any); ok {
		for k, _v := range val {
			if k == DescriptionKey {
				if meta.description == "" {
					if _desc, _ok := _v.(string); _ok {
						meta.description = _desc
					}
				}
				continue
			}
			fv, err := c.toNode(_v, metadata{})
			if err != nil {
				return node{}, err
			}
			res[k] = fv
		}
	} else {
		return node{}, errors.New("map must be of type map[string]any")
	}

	return node{
		meta:  meta,
		value: reflect.ValueOf(res),
	}, nil
}

func (c *cs) toNodeFromStruct(v any, meta metadata) (node, error) {
	// We assume this is a struct and convert this to a map of values
	res := map[string]node{}
	val := reflect.ValueOf(v)

	/*
		useValue := func(f reflect.Value) bool {

			switch {
			case !f.CanInterface():
				return false
			case f.Kind() == reflect.Ptr && f.IsNil():
				return false
					case f.Kind() == reflect.Slice && f.Len() == 0:
						return false

			default:
				return true
			}
		}

	*/

	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		// We skip this if the field is a zero value
		// TODO - allow this behavior to be configurable by source
		ft := val.Type().Field(i)
		if ft.Type == typeEntry {
			if meta.description == "" {
				meta.description = descriptionFromTag(ft.Tag)
			}
			// We skip if this is a nil pointer
		} else if f.CanInterface() &&
			// We can't have a null pointer to a struct or this loops forever
			!(f.Kind() == reflect.Ptr && f.IsNil() && f.Type().Elem().Kind() == reflect.Struct) { //nolint:staticcheck // won't use DeMorgan's law to allow shortcircuiting
			key := keyFromTag(ft.Tag)
			if key == "" {
				key = toLowerCamel(val.Type().Field(i).Name)
			}
			fv, err := c.toNode(f.Interface(),
				metadata{
					description: descriptionFromTag(ft.Tag),
				},
			)
			if err != nil {
				return node{}, err
			}
			res[key] = fv
		}
	}

	return node{
		meta:  meta,
		value: reflect.ValueOf(res),
	}, nil
}

func descriptionFromTag(tag reflect.StructTag) string {
	if desc, ok := tag.Lookup(DescriptionTagName); ok {
		return desc
	}
	return ""
}

func keyFromTag(tag reflect.StructTag) string {
	if key, ok := tag.Lookup(KeyTagName); ok {
		return key
	}
	return ""
}

func (c *cs) toNodeFromSlice(v any, meta metadata) (node, error) {
	var res []node
	val := reflect.ValueOf(v)
	t := reflect.TypeOf(v)

	for i := 0; i < val.Len(); i++ {
		_v := val.Index(i).Interface()
		fv, err := c.toNode(_v, metadata{})
		if err != nil {
			return node{}, err
		}
		res = append(res, fv)
	}

	// We translate the slice type two ways, if the underlying eleement type is a pointer, we use its element, if
	// the underlying type is astruct, we use the type of a mapAny
	ut := t.Elem()
	if ut.Kind() == reflect.Ptr {
		ut = ut.Elem()
	}
	if ut.Kind() == reflect.Struct {
		ut = typeMapStringAny
	}

	return node{
		meta:      meta,
		sliceType: reflect.SliceOf(ut),
		value:     reflect.ValueOf(res),
	}, nil
}

func (c *cs) fromNode(fullKey string, n node, into any) error {
	dest := reflect.ValueOf(into)
	if dest.Kind() == reflect.Ptr {
		dest = dest.Elem()
	}
	err := c.populateDestinationValue(fullKey, dest, n)
	if err != nil {
		return err
	}
	return c.validatingHook(into)
}

// populateDestinationValue populates the destination value based on its kind.
func (c *cs) populateDestinationValue(fullKey string, dest reflect.Value, n node) error {
	switch dest.Kind() {
	case reflect.Ptr:
		return c.populateDestinationPtr(fullKey, dest, n)
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		return c.populatePrimitiveValue(fullKey, dest, n)
	case reflect.Map:
		return c.populateMap(fullKey, dest, n)
	case reflect.Struct:
		return c.populateStruct(fullKey, dest, n)
	case reflect.Slice:
		return c.populateSlice(fullKey, dest, n)
	default:
		return fmt.Errorf("unsupported destination kind %s", dest.Kind().String())
	}
}

// populatePrimitiveValue populates a primitive value (string, int, float, bool) using late binding sources and type conversion.
func (c *cs) populatePrimitiveValue(fullKey string, dest reflect.Value, n node) error {
	for _, src := range c.lateBindingSources {
		lbVal, err := src(fullKey)
		if err != nil {
			return err
		}
		if lbVal != nil {
			n = node{value: reflect.ValueOf(lbVal)}
		}
	}

	// Skip setting if the value is zero
	if !n.value.IsValid() {
		return nil
	}

	// Type conversion, especially for late binding sources (e.g., environment variables as strings)
	err := c.castAndSet(dest, n.value)
	if err != nil {
		return err
	}

	return nil
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

func (c *cs) populateDestinationPtr(key string, dest reflect.Value, n node) error {

	// dest is always a ptr

	if !n.value.IsValid() {
		return nil
	}

	if dest.IsNil() {
		dest.Set(reflect.New(dest.Type().Elem()))
	}

	return c.populateDestinationValue(key, dest.Elem(), n)
}

func (c *cs) populateMap(fullKey string, dest reflect.Value, n node) error { //nolint:gocyclo // okay for marginally high complexity

	srcVal := n.value
	if srcVal.Kind() != reflect.Map {
		return nil // nothing to do if the source value is not a map
	}

	srcMap, ok := srcVal.Interface().(map[string]node)
	if !ok {
		return errors.New("invalid source map type. must be map[string]node")
	}
	if _, ok := dest.Interface().(map[string]any); !ok {
		return errors.New("invalid destination map type. must be map[string]any")
	}

	isSkippableValue := func(v reflect.Value) bool {
		if !v.IsValid() {
			return true
		}
		switch v.Kind() {
		case reflect.Map, reflect.Slice:
			return v.IsNil()
		default:
			return false
		}
	}

	joinKey := func(prefix, key string) string {
		if prefix == "" {
			return key
		}
		return fmt.Sprintf("%s.%s", prefix, key)
	}

	for srcKey, childNode := range srcMap {
		if isSkippableValue(childNode.value) {
			continue
		}

		pathKey := joinKey(fullKey, toLowerCamel(srcKey))
		destMapKey := reflect.ValueOf(srcKey) // keep original key for lookup/set

		existing := dest.MapIndex(destMapKey)
		if existing.IsValid() {
			if err := c.populateDestinationValue(pathKey, existing, childNode); err != nil {
				return err
			}
			continue
		}

		newValue := c.createDestinationValue(childNode)
		if err := c.populateDestinationValue(pathKey, newValue, childNode); err != nil {
			return err
		}
		dest.SetMapIndex(destMapKey, newValue)
	}

	return nil
}

// createDestinationValue creates a new destination value based on the source node's type.
func (c *cs) createDestinationValue(n node) reflect.Value {
	switch n.value.Kind() {
	// We can probably remove this case as it is dead code
	case reflect.Invalid:
		return reflect.Zero(reflect.PointerTo(n.nilType))
	default:
		switch n.value.Type() {
		case typeMapStringNode:
			return reflect.MakeMap(typeMapStringAny)
		case typeArrayNode:
			return reflect.New(n.sliceType).Elem()
		default:
			return reflect.New(n.value.Type()).Elem()
		}
	}
}

func (c *cs) populateStruct(fullKey string, dest reflect.Value, n node) error {

	// type must be map[string]reflect.Value
	if n.value.Kind() != reflect.Map {
		// Value is not a map, can't do anything
		return nil
	}

	// TODO - consider keys which are not present in the map

	if valMap, valMapOk := n.value.Interface().(map[string]node); valMapOk {

		for i := 0; i < dest.NumField(); i++ {
			f := dest.Field(i)
			ft := dest.Type().Field(i)
			key := keyFromTag(ft.Tag)
			if key == "" {
				key = toLowerCamel(dest.Type().Field(i).Name)
			}
			v := valMap[key]
			err := c.populateDestinationValue(fmt.Sprintf("%s.%s", fullKey, key), f, v)
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

func (c *cs) populateSlice(fullKey string, dest reflect.Value, n node) error {

	// type must be []reflect.Value
	if n.value.Kind() != reflect.Slice {
		// Value is not a slice, can't do anything
		return nil
	}

	if vals, valsOk := n.value.Interface().([]node); valsOk {
		for i, _val := range vals {

			var v reflect.Value
			vt := dest.Type().Elem()
			switch vt.Kind() {
			case reflect.Map:
				v = reflect.MakeMap(vt)
			default:
				v = reflect.New(vt).Elem()
			}

			err := c.populateDestinationValue(fmt.Sprintf("%s[%d]", fullKey, i), v, _val)
			if err != nil {
				return err
			}

			dest.Set(reflect.Append(dest, v))
		}

	} else {
		// Can't do anything, return nil
		return nil
	}

	return nil
}

func (c *cs) replaceOrMerge(existing node, in node) (node, error) { //nolint:gocyclo // okay for marginally high complexity

	switch existing.value.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		if in.value.Kind() == reflect.Map {
			return node{}, fmt.Errorf("cannot overrwrite type %s with a map", existing.value.Kind().String())
		}
		// Do nothing if the incoming value is not valid
		if in.value.IsZero() {
			return existing, nil
		}
		return in, nil
	// Right now we simply replace the slices
	case reflect.Slice:
		if in.value.Kind() != reflect.Slice {
			return node{}, fmt.Errorf("invalid value for slice target %s", in.value.Kind().String())
		}
		// Do nothing if the incoming value is not valid
		if in.value.IsZero() {
			return existing, nil
		}
		return in, nil
	case reflect.Map:
		if in.value.Kind() != reflect.Map {
			return node{}, fmt.Errorf("invalid value for map target %s", in.value.Kind().String())
		}
		// New must also be the same type
		if eMap, eOk := existing.value.Interface().(map[string]node); eOk {
			if nMap, nOk := in.value.Interface().(map[string]node); nOk {
				for k, v := range nMap {
					if el, elOk := eMap[k]; elOk {
						// map contains value, we merge
						var err error
						v, err = c.replaceOrMerge(el, v)
						if err != nil {
							return node{}, err
						}
					}
					eMap[k] = v
				}
			} else {
				return node{}, errors.New("new is unexpectedly not a map[string]node")
			}
			return existing, nil
		}
		return node{}, errors.New("destination is unexpectedly not a map[string]node")
	case reflect.Invalid:
		// This is the case for a nil value
		return in, nil
	default:
		return node{}, fmt.Errorf("unsupported existing kind %s", existing.value.Kind().String())
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

var (
	indexedKeyPattern = regexp.MustCompile(`^(\w+)\[([0-9]+)]$`)
)

func (c *cs) GetDescriptions(key string) (map[string]any, error) {

	res := map[string]any{}

	err := c.withCleanData(func() error {
		err := c.read(key, key, c.root, &res)
		if err != nil {
			err = wrapError(&res, key, err)
		}
		return err
	})

	if err != nil {
		return nil, err
	}

	return res, nil

}

func (c *cs) read(fullKey, key string, data map[string]node, into any) error {
	parts := strings.SplitN(key, ".", 2)
	thisKey := parts[0]
	if thisKey == "" {
		// Special case for root of the cs
		return c.fromNode("", node{value: reflect.ValueOf(c.root)}, into)
	}
	i := -1
	if matches := indexedKeyPattern.FindAllStringSubmatch(thisKey, -1); matches != nil {
		i, _ = strconv.Atoi(matches[0][2])
		thisKey = matches[0][1]
	}
	if tmp, ok := data[thisKey]; ok {
		if i >= 0 {
			// This doesn't seem right but it seems to work
			tmp = tmp.value.Index(i).Interface().(node)
		}
		if len(parts) == 1 {
			return c.fromNode(fullKey, tmp, into)
		} else if data, ok = tmp.value.Interface().(map[string]node); ok {
			return c.read(fullKey, parts[1], data, into)
		}
		return fmt.Errorf("invalid type for key %s", thisKey)
	}
	// We still populate the value in the case it is a struct and we can lookup keys based on fields
	return c.fromNode(fullKey, node{value: reflect.New(typeMapStringNode).Elem()}, into)
}

func (c *cs) Read(key string, into any) error {
	if reflect.ValueOf(into).Kind() != reflect.Ptr {
		return errors.New("into must be a pointer")
	}
	return c.withCleanData(func() error {
		err := c.read(key, key, c.root, into)
		if err != nil {
			err = wrapError(into, key, err)
		}
		return err
	})
}

func (c *cs) MustRead(key string, into any) {
	if err := c.Read(key, into); err != nil {
		panic(err)
	}
}

func (c *cs) MustGetDescriptions(key string) map[string]any {
	res, err := c.GetDescriptions(key)
	if err != nil {
		panic(err)
	}
	return res
}

func newConfig() Config {
	return &cs{
		root: map[string]node{},
		validatingHook: func(in any) error {
			if v, ok := in.(Validating); ok {
				return v.Validate()
			}
			return nil
		},
	}
}
