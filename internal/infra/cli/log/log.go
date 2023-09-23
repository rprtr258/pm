package log

import (
	"bytes"
	"cmp"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/rprtr258/pm/internal/infra/cli/log/buffer"
)

func colorStringFg(bb []byte, color []byte) []byte {
	return []byte(buffer.NewString(func(b *buffer.Buffer) {
		b.Styled(func(b *buffer.Buffer) {
			b.Bytes(bb...)
		}, color)
	}))
}

type prettyWriter struct {
	maxSlicePrintSize int
	b                 *buffer.Buffer
}

func levelColors(level zerolog.Level) (bg, fg []byte) { //nolint:nonamedreturns // for documentation purposes
	switch {
	case level < zerolog.InfoLevel:
		return buffer.BgBlue, buffer.FgBlue
	case level < zerolog.WarnLevel:
		return buffer.BgGreen, buffer.FgGreen
	case level < zerolog.ErrorLevel:
		return buffer.BgYellow, buffer.FgYellow
	default:
		return buffer.BgRed, buffer.FgRed
	}
}

func iterSorted(seq iter.Seq[fun.Pair[string, any]]) iter.Seq[fun.Pair[string, any]] {
	slice := seq.ToSlice()
	slices.SortFunc(slice, func(i, j fun.Pair[string, any]) int {
		return cmp.Compare(i.K, j.K)
	})
	return iter.FromMany(slice...)
}

func isURL(u string) bool {
	_, err := url.ParseRequestURI(u)
	return err == nil
}

func (w *prettyWriter) formatSlice(st reflect.Type, sv reflect.Value, l int) []byte {
	d := min(len(strconv.Itoa(sv.Len())), len(strconv.Itoa(w.maxSlicePrintSize)))

	var bb bytes.Buffer
	buffer.New(&bb).
		Bytes(w.buildTypeString(st.String())...).
		InBytePair('(', ')', func(b *buffer.Buffer) {
			b.String(strconv.Itoa(sv.Len()), buffer.FgBlue)
		}).
		Iter(iter.Map(iter.FromRange(0, sv.Len(), 1), func(i int) func(*buffer.Buffer) {
			return func(b *buffer.Buffer) {
				if i == w.maxSlicePrintSize {
					b.
						Bytes('\n').
						RepeatByte(' ', l*2+4).
						RepeatByte(' ', d+2).
						String("...", buffer.FgBlue).
						Styled(func(b *buffer.Buffer) {
							b.Bytes(']')
						}, buffer.FgGreen)
					return
				}

				v := sv.Index(i)
				tb := strconv.Itoa(i)
				b.
					Bytes('\n').
					RepeatByte(' ', l*2+4).
					RepeatByte(' ', d-len(tb)).
					String(tb, buffer.FgGreen).
					Bytes(' ').
					Bytes(w.formatValue(v, l+1)...)
			}
		}))
	return bb.Bytes()
}

func (w *prettyWriter) formatMap(typ reflect.Type, val reflect.Value, l int) []byte {
	p := 0
	for _, k := range val.MapKeys() {
		p = max(p, len(anyToBytes(k)))
	}
	p += len(buffer.FgGreen) + len(buffer.ColorReset)

	sk := val.MapKeys()
	slices.SortFunc(sk, func(i, j reflect.Value) int {
		return cmp.Compare(fmt.Sprint(i.Interface()), fmt.Sprint(j.Interface()))
	})

	var bb bytes.Buffer
	buffer.New(&bb).
		Bytes(w.buildTypeString(typ.String())...).
		InBytePair('(', ')', func(b *buffer.Buffer) {
			b.String(strconv.Itoa(val.Len()), buffer.FgBlue)
		}).
		Iter(iter.Map(iter.FromMany(sk...), func(k reflect.Value) func(*buffer.Buffer) {
			return func(b *buffer.Buffer) {
				tb := colorStringFg(w.formatValue(k, l+1), buffer.FgGreen)
				b.
					Bytes('\n').
					RepeatByte(' ', l*2+4).
					Bytes(tb...).
					RepeatByte(' ', p-len(tb)).
					Bytes(' ').
					Bytes(w.formatValue(val.MapIndex(k), l+1)...)
			}
		}))
	return bb.Bytes()
}

func (w *prettyWriter) formatStruct(st reflect.Type, sv reflect.Value, l int) []byte {
	p := 0
	for i := 0; i < st.NumField(); i++ {
		p = max(p, len(st.Field(i).Name))
	}
	p += len(buffer.FgGreen) + len(buffer.ColorReset)

	zeroes := 0
	var bb bytes.Buffer
	buffer.New(&bb).
		Bytes(w.buildTypeString(st.String())...).
		Iter(iter.Map(iter.FromRange(0, st.NumField(), 1), func(i int) func(*buffer.Buffer) {
			return func(b *buffer.Buffer) {
				val := sv.Field(i)
				if val.IsZero() {
					zeroes++
					return
				}

				fieldName := colorStringFg([]byte(sv.Type().Field(i).Name), buffer.FgGreen)

				b.
					Bytes('\n').
					RepeatByte(' ', l*2+4).
					Bytes(fieldName...).
					RepeatByte(' ', p-len(fieldName)).
					Bytes(' ').
					Bytes(w.formatValue(val, l+1)...)
			}
		})).
		Styled(func(b *buffer.Buffer) {
			if zeroes > 0 {
				b.
					Bytes('\n').
					RepeatByte(' ', l*2+4).
					String("// zeros", buffer.ColorFaint)
			}
		})
	return bb.Bytes()
}

func (w *prettyWriter) formatValue(v reflect.Value, l int) []byte {
	if v.IsZero() {
		var bb bytes.Buffer
		buffer.New(&bb).
			String(fmt.Sprint(v.Interface()), buffer.ColorFaint)
		return bb.Bytes()
	}

	var res []byte
	switch t := v.Type(); t.Kind() { //nolint:exhaustive // not needed
	case reflect.Slice:
		return w.formatSlice(t, v, l)
	case reflect.Map:
		return w.formatMap(t, v, l)
	case reflect.Struct:
		res = w.formatStruct(t, v, l)
	case reflect.Interface:
		if !v.IsZero() {
			res = w.formatValue(v.Elem(), l)
		} else {
			res = []byte("nil")
		}
	case reflect.Pointer:
		for v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		res = w.formatValue(v, l)
	default:
		res = anyToBytes(v)
	}

	if s := fmt.Sprint(v.Interface()); s == "<nil>" || s == "0" || s == "false" {
		return []byte(buffer.NewString(func(b *buffer.Buffer) {
			b.Styled(func(b *buffer.Buffer) {
				b.Bytes(res...)
			}, buffer.FgWhite, buffer.ColorFaint)
		}))
	}

	return res
}

func (w *prettyWriter) buildTypeString(typeStr string) []byte {
	typeStr = strings.ReplaceAll(typeStr, "interface {}", "any")
	var bb bytes.Buffer
	buffer.New(&bb).
		Iter(iter.Map(iter.FromMany([]byte(typeStr)...), func(c byte) func(*buffer.Buffer) {
			return func(b *buffer.Buffer) {
				b.
					Bytes(lo.Switch[byte, []byte](c).
						Case('*', buffer.FgRed).
						Case('[', buffer.FgGreen).
						Case(']', buffer.FgGreen).
						Default(buffer.FgYellow)...).
					Bytes(c)
			}
		})).
		Bytes(buffer.ColorReset...)
	return bb.Bytes()
}

// anyToBytes using fmt.Sprint
func anyToBytes(a reflect.Value) []byte {
	return []byte(fmt.Sprint(a.Interface()))
}

func (w *prettyWriter) write(msg string, ev *Event) {
	colorBg, colorFg := levelColors(ev.level)

	padding := 0
	for k := range ev.fields {
		padding = max(padding, len(k))
	}
	padding += len(buffer.FgMagenta) + len(buffer.ColorReset)

	w.b.
		String(ev.ts.Format("[15:06:05]"), buffer.ColorFaint, buffer.FgWhite).
		Bytes(' ').
		// level
		Styled(func(b *buffer.Buffer) {
			b.InBytePair(' ', ' ', func(b *buffer.Buffer) {
				b.String(strings.ToUpper(ev.level.String()))
			})
		}, buffer.FgBlack, colorBg).
		Bytes(' ').
		String(msg, colorFg).
		Bytes('\n').
		// attributes
		Iter(iter.Map(iterSorted(iter.FromDict(ev.fields)), func(kv fun.Pair[string, any]) func(*buffer.Buffer) {
			k, value := kv.K, kv.V
			return func(b *buffer.Buffer) {
				b.
					String(k, buffer.FgMagenta).
					RepeatByte(' ', padding-len(k)).
					Bytes(' ')
				switch vv := value.(type) {
				case time.Time, time.Duration:
					b.String(fmt.Sprint(vv), buffer.FgCyan)
				case *time.Time:
					b.String(vv.String(), buffer.FgCyan)
				case *time.Duration:
					b.String(vv.String(), buffer.FgCyan)
				default:
					at := reflect.TypeOf(value)
					av := reflect.ValueOf(value)
					switch at.Kind() { //nolint:exhaustive // not needed
					case reflect.Float32, reflect.Float64,
						reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
						reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						b.String(fmt.Sprint(value), buffer.FgYellow)
					case reflect.Bool:
						b.String(fmt.Sprint(value), buffer.FgRed)
					case reflect.String:
						v := value.(string) //nolint:forcetypeassert,errcheck // checked kind already
						switch {
						case v == "":
							b.String("empty", buffer.FgWhite, buffer.ColorFaint)
						case isURL(v):
							b.String(v, buffer.FgBlue, buffer.ColorUnderline)
						default:
							b.String(v)
						}
					case reflect.Pointer:
						for av.Kind() == reflect.Pointer {
							av = av.Elem()
						}
						b.Bytes(w.formatValue(av, 0)...)
					case reflect.Slice, reflect.Array:
						b.Bytes(w.formatSlice(at, av, 0)...)
					case reflect.Map:
						b.Bytes(w.formatMap(at, av, 0)...)
					case reflect.Struct:
						b.Bytes(w.formatStruct(at, av, 0)...)
					default:
						b.String(fmt.Sprint(value))
					}
				}
				b.Bytes('\n')
			}
		}))
}

type Event struct {
	ts     time.Time
	level  zerolog.Level
	fields map[string]any
}

func (e *Event) Str(k string, v string) *Event {
	e.fields[k] = v
	return e
}

func (e *Event) Time(k string, v time.Time) *Event {
	e.fields[k] = v
	return e
}

func (e *Event) Any(k string, v any) *Event {
	e.fields[k] = v
	return e
}

func (e *Event) Err(err error) *Event {
	e.fields[zerolog.ErrorFieldName] = err
	return e
}

func (e *Event) Errs(k string, errs []error) *Event {
	e.fields[k] = errs
	return e
}

func (e *Event) Msg(msg string) {
	(&prettyWriter{
		maxSlicePrintSize: 10,
		b:                 buffer.New(os.Stderr),
	}).write(msg, e)
}

type logger struct{}

var _logger = &logger{}

func (l *logger) log(level zerolog.Level) *Event {
	return &Event{
		ts:     time.Now(),
		level:  level,
		fields: map[string]any{},
	}
}

func (l *logger) Info() *Event {
	return l.log(zerolog.InfoLevel)
}

func (l *logger) Warn() *Event {
	return l.log(zerolog.WarnLevel)
}

func (l *logger) Error() *Event {
	return l.log(zerolog.ErrorLevel)
}

func (l *logger) Fatal(err error) {
	e := l.log(zerolog.FatalLevel)
	if xErr, ok := err.(*xerr.Error); ok { //nolint:nestif // ok, fuck you
		if xErr.Message != "" {
			e.Str("msg", xErr.Message)
		}
		if xErr.Err != nil {
			e.Err(xErr.Err)
		}
		if len(xErr.Errs) > 0 {
			e.Errs("errs", xErr.Errs)
		}
		if !xErr.At.IsZero() {
			e.Time("at", xErr.At)
		}
		if caller := xErr.Caller; caller != nil {
			e.Str("err_caller", fmt.Sprintf("%s:%d#%s", caller.File, caller.Line, caller.Function))
		}
		for i, frame := range xErr.Stacktrace {
			e.Str(fmt.Sprintf("stack[%d]", i), fmt.Sprintf("%s:%d#%s", frame.File, frame.Line, frame.Function))
		}
		for k, v := range xErr.Fields {
			e.Any(k, v)
		}
	} else {
		e = e.Err(err)
	}

	e.Msg("app exited abnormally")
	os.Exit(1)
}

func Info() *Event {
	return _logger.Info()
}

func Warn() *Event {
	return _logger.Warn()
}

func Error() *Event {
	return _logger.Error()
}

func Fatal(err error) {
	_logger.Fatal(err)
}
