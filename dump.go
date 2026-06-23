package cs

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
)

// dumpOps provides a way to defer dump operations, allowing parents to know if they are going to dump as well
type dumpOps []func() error

func (d dumpOps) dump() error {
	for _, o := range d {
		if err := o(); err != nil {
			return err
		}
	}
	return nil
}

type dumpOptions struct {
	out       io.Writer
	omitEmpty bool
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

// WithOmitEmpty instructs ths dumper to not show empty entries
func WithOmitEmpty() DumpOption {
	return func(o *dumpOptions) {
		o.omitEmpty = true
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
		op, err := c.dumpNode("", data, &dumpContext{
			opts:   o,
			level:  0,
			prefix: "",
		})
		if err != nil {
			return err
		}
		if op == nil {
			return nil
		}
		return op()
	})

}

func (c *cs) overlayLateBinding(fullKey string, val node) (node, error) {

	switch val.getEffectiveType().Kind() {
	case reflect.Map:
		tmp, err := c.overlayLateBindingMap(fullKey, val.value.Interface().(map[string]node))
		if err != nil {
			return node{}, err
		}
		return node{
			meta:      val.meta,
			sliceType: val.sliceType,
			value:     reflect.ValueOf(tmp),
		}, nil
	case reflect.Slice:
		tmp, err := c.overlayLateBindingSlice(fullKey, val.value.Interface().([]node))
		if err != nil {
			return node{}, err
		}
		return node{
			meta:      val.meta,
			sliceType: val.sliceType,
			value:     reflect.ValueOf(tmp),
		}, nil
	default:
		var lbRes any
		// Locked keys ignore late-binding (env) sources in the dump, just as
		// they do at read time, so the dump shows the locked value.
		if !c.isLocked(fullKey) {
			for _, lbs := range c.lateBindingSources {
				lbVal, err := lbs(fullKey)
				if err != nil {
					return node{}, err
				}
				if lbVal != nil {
					lbRes = lbVal
				}
			}
		}
		if lbRes != nil {

			tmp := reflect.New(val.value.Type()).Elem()
			err := c.castAndSet(tmp, reflect.ValueOf(lbRes))
			if err != nil {
				return node{}, err
			}
			return node{
				meta:      val.meta,
				sliceType: val.sliceType,
				value:     tmp,
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
	if val.meta.description != "" {
		sb.WriteString(fmt.Sprintf(" %s", val.meta.description))
	}
	if val.meta.optional {
		sb.WriteString(" [optional]")
	}
	if val.meta.locked {
		sb.WriteString(" [locked]")
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
	if val.meta.description != "" {
		sb.WriteString(fmt.Sprintf(" %s", val.meta.description))
	}
	if val.meta.optional {
		sb.WriteString(" [optional]")
	}
	if val.meta.locked {
		sb.WriteString(" [locked]")
	}

	outVal := val.interfaceOrNil()

	if outVal != nil {
		sb.WriteString(" (")
		sb.WriteString(fmt.Sprintf("%v", outVal))
		sb.WriteString(")")
	}
	sb.WriteString("\n")
	_, err := ctx.opts.out.Write([]byte(sb.String()))
	return err
}

func (c *cs) dumpNodeMap(val map[string]node, ctx *dumpContext) (dumpOps, error) {

	keys := make([]string, len(val))
	i := 0
	for k := range val {
		keys[i] = k
		i++
	}

	slices.Sort(keys)
	var ops dumpOps

	for _, k := range keys {
		op, err := c.dumpNode(k, val[k], ctx)
		if err != nil {
			return nil, err
		}
		if op != nil {
			ops = append(ops, op)
		}
	}

	return ops, nil
}

func (c *cs) dumpNodeSlice(val []node, ctx *dumpContext) (dumpOps, error) {

	var ops dumpOps

	for i, v := range val {
		op, err := c.dumpNode(fmt.Sprintf("[%d]", i), v, ctx)
		if err != nil {
			return nil, err
		}
		if op != nil {
			ops = append(ops, op)
		}
	}

	return ops, nil
}

func (c *cs) dumpNode(name string, val node, ctx *dumpContext) (func() error, error) { //nolint:gocyclo // marginally high is okay

	switch val.getEffectiveType().Kind() {
	case reflect.Map:
		// We don't write header line for the root
		addLevel := 0
		if name != "" {
			addLevel = 1
		}
		ops, err := c.dumpNodeMap(val.value.Interface().(map[string]node), ctx.modify(dumpModify{
			appendPrefix: name,
			addLevel:     addLevel,
		}))
		if err != nil {
			return nil, err
		}
		if len(ops) != 0 || !ctx.opts.omitEmpty {
			return func() error {
				// special case for name is blank
				if name != "" {
					err = c.writeHeaderLine(name, val, ctx)
					if err != nil {
						return err
					}
				}
				return ops.dump()
			}, nil
		}
	case reflect.Slice:
		ops, err := c.dumpNodeSlice(val.value.Interface().([]node), ctx.modify(dumpModify{
			appendPrefix: name,
			addLevel:     1,
		}))
		if err != nil {
			return nil, err
		}
		if len(ops) != 0 || !ctx.opts.omitEmpty {
			return func() error {
				err = c.writeHeaderLine(name, val, ctx)
				if err != nil {
					return err
				}
				return ops.dump()
			}, nil
		}
	default:
		if (val.value.IsValid() && !val.value.IsZero()) || !ctx.opts.omitEmpty {
			return func() error {
				return c.writeLine(name, val, ctx)
			}, nil
		}
	}

	return nil, nil
}
