package cs

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
)

type dumpOptions struct {
	out io.Writer
}

type dumpContext struct {
	opts   *dumpOptions
	level  int
	prefix string
}

type dumpModify struct {
	appendPrefix string
	addLevel     int
}

func (c *dumpContext) modify(m dumpModify) *dumpContext {
	res := &dumpContext{
		opts:  c.opts,
		level: c.level + m.addLevel,
	}
	if m.appendPrefix != "" {
		res.prefix = fmt.Sprintf("%s.%s", c.prefix, m.appendPrefix)
	}
	return res
}

// DumpOption represents a functional option used to configure the behavior of the dump operation.
type DumpOption func(o *dumpOptions)

// WithDumpOut sets a custom io.Writer as the output destination for the dump process.
func WithDumpOut(out io.Writer) DumpOption {
	return func(o *dumpOptions) {
		o.out = out
	}
}

// Dump outputs the internal state of the structure to an io.Writer, such as os.Stdout, with customizable options.
func (c *cs) Dump(opts ...DumpOption) error {

	o := &dumpOptions{
		out: os.Stdout,
	}

	for _, opt := range opts {
		opt(o)
	}

	return c.withCleanData(func() error {
		data, err := c.overlayLateBinding("", node{
			value: reflect.ValueOf(c.root),
		})
		if err != nil {
			return err
		}
		return c.dumpNode("", data, &dumpContext{
			opts:   o,
			level:  0,
			prefix: "",
		})
	})

}

func (c *cs) overlayLateBinding(fullKey string, val node) (node, error) {

	switch val.value.Type().Kind() {
	case reflect.Map:
		tmp, err := c.overlayLateBindingMap(fullKey, val.value.Interface().(map[string]node))
		if err != nil {
			return node{}, err
		}
		return node{
			description: val.description,
			sliceType:   val.sliceType,
			value:       reflect.ValueOf(tmp),
		}, nil
	case reflect.Slice:
		tmp, err := c.overlayLateBindingSlice(fullKey, val.value.Interface().([]node))
		if err != nil {
			return node{}, err
		}
		return node{
			description: val.description,
			sliceType:   val.sliceType,
			value:       reflect.ValueOf(tmp),
		}, nil
	default:
		var lbRes any
		for _, lbs := range c.lateBindingSources {
			lbVal, err := lbs(fullKey)
			if err != nil {
				return node{}, err
			}
			if lbVal != nil {
				lbRes = lbVal
			}
		}
		if lbRes != nil {

			tmp := reflect.New(val.value.Type()).Elem()
			err := c.castAndSet(tmp, reflect.ValueOf(lbRes))
			if err != nil {
				return node{}, err
			}
			return node{
				description: val.description,
				sliceType:   val.sliceType,
				value:       tmp,
			}, nil
		}
		return val, nil
	}
}

func (c *cs) overlayLateBindingMap(fullKey string, val map[string]node) (map[string]node, error) {
	res := map[string]node{}
	for k, v := range val {
		_fmt := "%s.%s"
		if fullKey == "" {
			_fmt = "%s%s"
		}
		tmp, err := c.overlayLateBinding(fmt.Sprintf(_fmt, fullKey, k), v)
		if err != nil {
			return nil, err
		}
		res[k] = tmp
	}
	return res, nil
}

func (c *cs) overlayLateBindingSlice(fullKey string, val []node) ([]node, error) {
	res := make([]node, len(val))
	for i, v := range val {
		tmp, err := c.overlayLateBinding(fmt.Sprintf("%s[%d]", fullKey, i), v)
		if err != nil {
			return nil, err
		}
		res[i] = tmp
	}
	return res, nil
}

func (c *cs) writeHeaderLine(name string, val node, ctx *dumpContext) error {
	sb := strings.Builder{}
	if ctx.level > 0 {
		sb.WriteString(strings.Repeat("  ", ctx.level))
	}
	sb.WriteString(fmt.Sprintf("%s:", name))
	if val.description != "" {
		sb.WriteString(fmt.Sprintf(" %s", val.description))
	}
	sb.WriteString("\n")
	_, err := ctx.opts.out.Write([]byte(sb.String()))
	return err
}

func (c *cs) writeLine(name string, val node, ctx *dumpContext) error {
	sb := strings.Builder{}
	if ctx.level > 0 {
		sb.WriteString(strings.Repeat("  ", ctx.level))
	}
	sb.WriteString(fmt.Sprintf("%s:", name))
	if val.description != "" {
		sb.WriteString(fmt.Sprintf(" %s", val.description))
	}

	sb.WriteString(" (")
	sb.WriteString(fmt.Sprintf("%v", val.value.Interface()))
	sb.WriteString(")\n")
	_, err := ctx.opts.out.Write([]byte(sb.String()))
	return err
}

func (c *cs) dumpNodeMap(val map[string]node, ctx *dumpContext) error {

	var err error

	keys := make([]string, len(val))
	i := 0
	for k := range val {
		keys[i] = k
		i++
	}

	slices.Sort(keys)

	for _, k := range keys {
		err = c.dumpNode(k, val[k], ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *cs) dumpNodeSlice(val []node, ctx *dumpContext) error {

	var err error

	for i, v := range val {
		err = c.dumpNode(fmt.Sprintf("[%d]", i), v, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *cs) dumpNode(name string, val node, ctx *dumpContext) error {

	var err error

	switch val.value.Type().Kind() {
	case reflect.Map:
		// We don't write header line for the root
		addLevel := 0
		if name != "" {
			err = c.writeHeaderLine(name, val, ctx)
			if err != nil {
				return err
			}
			addLevel = 1
		}
		return c.dumpNodeMap(val.value.Interface().(map[string]node), ctx.modify(dumpModify{
			appendPrefix: name,
			addLevel:     addLevel,
		}))
	case reflect.Slice:
		err = c.writeHeaderLine(name, val, ctx)
		if err != nil {
			return err
		}
		return c.dumpNodeSlice(val.value.Interface().([]node), ctx.modify(dumpModify{
			appendPrefix: name,
			addLevel:     1,
		}))
	default:
		err = c.writeLine(name, val, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
