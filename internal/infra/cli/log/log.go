package log

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
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
)

//nolint:unused // fuck you
var (
	// Foreground colors
	fgBlack   = []byte("\x1b[30m")
	fgRed     = []byte("\x1b[31m")
	fgGreen   = []byte("\x1b[32m")
	fgYellow  = []byte("\x1b[33m")
	fgBlue    = []byte("\x1b[34m")
	fgMagenta = []byte("\x1b[35m")
	fgCyan    = []byte("\x1b[36m")
	fgWhite   = []byte("\x1b[37m")

	// Background colors
	bgBlack   = []byte("\x1b[40m")
	bgRed     = []byte("\x1b[41m")
	bgGreen   = []byte("\x1b[42m")
	bgYellow  = []byte("\x1b[43m")
	bgBlue    = []byte("\x1b[44m")
	bgMagenta = []byte("\x1b[45m")
	bgCyan    = []byte("\x1b[46m")
	bgWhite   = []byte("\x1b[47m")

	// Common consts
	resetColor     = []byte("\x1b[0m")
	faintColor     = []byte("\x1b[2m")
	underlineColor = []byte("\x1b[4m")
)

type buffer struct {
	out io.Writer
}

func newBuffer(out io.Writer) *buffer {
	return &buffer{out}
}

func (b *buffer) bytes(bs ...byte) *buffer {
	b.out.Write(bs) //nolint:errcheck // fuck you
	return b
}

func (b *buffer) repeatByte(c byte, n int) *buffer { //nolint:unparam // fuck you
	b.out.Write(bytes.Repeat([]byte{c}, n)) //nolint:errcheck // fuck you
	return b
}

func (b *buffer) string(s string) *buffer {
	io.WriteString(b.out, s) //nolint:errcheck // fuck you
	return b
}

func (b *buffer) styled(f func(*buffer), mods ...[]byte) *buffer {
	for _, mod := range mods {
		b.out.Write(mod) //nolint:errcheck // fuck you
	}
	f(b)
	b.out.Write(resetColor) //nolint:errcheck // fuck you
	return b
}

func (b *buffer) inside(start, end byte, f func(*buffer)) *buffer {
	return b.styled(func(b *buffer) {
		b.bytes(start)
		f(b)
		b.bytes(end)
	})
}

func (b *buffer) iter(seq iter.Seq[func(*buffer)]) *buffer {
	seq(func(f func(*buffer)) bool {
		f(b)
		return true
	})
	return b
}

func newString(f func(*buffer)) string {
	var bb bytes.Buffer
	f(newBuffer(&bb))
	return bb.String()
}

func colorStringFg(bb []byte, color []byte) []byte {
	return []byte(newString(func(b *buffer) {
		b.styled(func(b *buffer) {
			b.bytes(bb...)
		}, color)
	}))
}

type prettyWriter struct {
	maxSlicePrintSize int
	b                 *buffer
}

func levelColors(level zerolog.Level) (colorBg, colorFg []byte) { //nolint:nonamedreturns // for documentation purposes
	switch {
	case level < zerolog.InfoLevel:
		return bgBlue, fgBlue
	case level < zerolog.WarnLevel:
		return bgGreen, fgGreen
	case level < zerolog.ErrorLevel:
		return bgYellow, fgYellow
	default:
		return bgRed, fgRed
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
	newBuffer(&bb).
		bytes(w.buildTypeString(st.String())...).
		inside('(', ')', func(b *buffer) {
			b.styled(func(b *buffer) {
				b.string(strconv.Itoa(sv.Len()))
			}, fgBlue)
		}).
		iter(iter.Map(iter.FromRange(0, sv.Len(), 1), func(i int) func(*buffer) {
			return func(b *buffer) {
				if i == w.maxSlicePrintSize {
					b.
						bytes('\n').
						repeatByte(' ', l*2+4).
						repeatByte(' ', d+2).
						styled(func(b *buffer) {
							b.string("...")
						}, fgBlue).
						styled(func(b *buffer) {
							b.bytes(']')
						}, fgGreen)
					return
				}

				v := sv.Index(i)
				tb := strconv.Itoa(i)
				b.
					bytes('\n').
					repeatByte(' ', l*2+4).
					repeatByte(' ', d-len(tb)).
					styled(func(b *buffer) {
						b.string(tb)
					}, fgGreen).
					bytes(' ').
					bytes(w.formatValue(v, l)...)
			}
		}))
	return bb.Bytes()
}

func (w *prettyWriter) formatMap(typ reflect.Type, val reflect.Value, l int) []byte {
	p := 0
	for _, k := range val.MapKeys() {
		p = max(p, len(anyToBytes(k)))
	}
	p += len(fgGreen) + len(resetColor)

	sk := val.MapKeys()
	slices.SortFunc(sk, func(i, j reflect.Value) int {
		return cmp.Compare(fmt.Sprint(i.Interface()), fmt.Sprint(j.Interface()))
	})

	var bb bytes.Buffer
	newBuffer(&bb).
		bytes(w.buildTypeString(typ.String())...).
		inside('(', ')', func(b *buffer) {
			b.styled(func(b *buffer) {
				b.string(strconv.Itoa(val.Len()))
			}, fgBlue)
		}).
		iter(iter.Map(iter.FromMany(sk...), func(k reflect.Value) func(*buffer) {
			return func(b *buffer) {
				tb := colorStringFg(w.formatValue(k, l), fgGreen)
				b.
					bytes('\n').
					repeatByte(' ', l*2+4).
					bytes(tb...).
					repeatByte(' ', p-len(tb)).
					bytes(' ').
					bytes(w.formatValue(val.MapIndex(k), l)...)
			}
		}))
	return bb.Bytes()
}

func (w *prettyWriter) formatStruct(st reflect.Type, sv reflect.Value, l int) []byte {
	p := 0
	for i := 0; i < st.NumField(); i++ {
		p = max(p, len(st.Field(i).Name))
	}
	p += len(fgGreen) + len(resetColor)

	zeroes := 0
	var bb bytes.Buffer
	newBuffer(&bb).
		bytes(w.buildTypeString(st.String())...).
		iter(iter.Map(iter.FromRange(0, st.NumField(), 1), func(i int) func(*buffer) {
			return func(b *buffer) {
				val := sv.Field(i)
				if val.IsZero() {
					zeroes++
					return
				}

				fieldName := colorStringFg([]byte(sv.Type().Field(i).Name), fgGreen)

				b.
					bytes('\n').
					repeatByte(' ', l*2+4).
					bytes(fieldName...).
					repeatByte(' ', p-len(fieldName)).
					bytes(' ').
					bytes(w.formatValue(val, l)...)
			}
		})).
		styled(func(b *buffer) {
			if zeroes > 0 {
				b.
					bytes('\n').
					repeatByte(' ', l*2+4).
					string("// zeros")
			}
		}, faintColor)
	return bb.Bytes()
}

func (w *prettyWriter) formatValue(v reflect.Value, l int) []byte {
	if v.IsZero() {
		var bb bytes.Buffer
		newBuffer(&bb).styled(func(b *buffer) {
			b.string(fmt.Sprint(v.Interface()))
		}, faintColor)
		return bb.Bytes()
	}

	var res []byte
	switch t := v.Type(); t.Kind() { //nolint:exhaustive // not needed
	case reflect.Slice:
		return w.formatSlice(t, v, l+1)
	case reflect.Map:
		return w.formatMap(t, v, l+1)
	case reflect.Struct:
		res = w.formatStruct(t, v, l+1)
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
		return []byte(newString(func(b *buffer) {
			b.styled(func(b *buffer) {
				b.bytes(res...)
			}, fgWhite, faintColor)
		}))
	}

	return res
}

func (w *prettyWriter) buildTypeString(typeStr string) []byte {
	typeStr = strings.ReplaceAll(typeStr, "interface {}", "any")
	var bb bytes.Buffer
	newBuffer(&bb).
		iter(iter.Map(iter.FromMany([]byte(typeStr)...), func(c byte) func(*buffer) {
			return func(b *buffer) {
				b.
					bytes(lo.Switch[byte, []byte](c).
						Case('*', fgRed).
						Case('[', fgGreen).
						Case(']', fgGreen).
						Default(fgYellow)...).
					bytes(c)
			}
		})).
		bytes(resetColor...)
	return bb.Bytes()
}

// anyToBytes using fmt.Sprint
func anyToBytes(a reflect.Value) []byte {
	return []byte(fmt.Sprint(a.Interface()))
}

func (w *prettyWriter) write(msg string, ev *event) { //nolint:funlen // fuck you
	colorBg, colorFg := levelColors(ev.level)

	padding := 0
	for k := range ev.fields {
		padding = max(padding, len(k))
	}
	padding += len(fgMagenta) + len(resetColor)

	w.b.
		styled(func(b *buffer) {
			b.string(ev.ts.Format("[15:06:05]"))
		}, faintColor, fgWhite).
		bytes(' ').
		// level
		styled(func(b *buffer) {
			b.inside(' ', ' ', func(b *buffer) {
				b.string(strings.ToUpper(ev.level.String()))
			})
		}, fgBlack, colorBg).
		bytes(' ').
		styled(func(b *buffer) {
			b.string(msg)
		}, colorFg).
		bytes('\n').
		// attributes
		iter(iter.Map(iterSorted(iter.FromDict(ev.fields)), func(kv fun.Pair[string, any]) func(*buffer) {
			k, value := kv.K, kv.V
			return func(b *buffer) {
				b.
					styled(func(b *buffer) {
						b.string(k)
					}, fgMagenta).
					repeatByte(' ', padding-len(k)).
					bytes(' ')
				switch vv := value.(type) {
				case time.Time, time.Duration:
					b.styled(func(b *buffer) {
						b.string(fmt.Sprint(vv))
					}, fgCyan)
				case *time.Time:
					b.styled(func(b *buffer) {
						b.string(vv.String())
					}, fgCyan)
				case *time.Duration:
					b.styled(func(b *buffer) {
						b.string(vv.String())
					}, fgCyan)
				default:
					at := reflect.TypeOf(value)
					av := reflect.ValueOf(value)
					switch at.Kind() { //nolint:exhaustive // not needed
					case reflect.Float32, reflect.Float64,
						reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
						reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						b.styled(func(b *buffer) {
							b.string(fmt.Sprint(value))
						}, fgYellow)
					case reflect.Bool:
						b.styled(func(b *buffer) {
							b.string(fmt.Sprint(value))
						}, fgRed)
					case reflect.String:
						v := value.(string) //nolint:forcetypeassert,errcheck // checked kind already
						switch {
						case v == "":
							b.styled(func(b *buffer) {
								b.string("empty")
							}, fgWhite, faintColor)
						case isURL(v):
							b.styled(func(b *buffer) {
								b.string(v)
							}, fgBlue, underlineColor)
						default:
							b.string(v)
						}
					case reflect.Pointer:
						for av.Kind() == reflect.Pointer {
							av = av.Elem()
						}
						b.bytes(w.formatValue(av, -1)...) // TODO: remove kostyl with -1
					case reflect.Slice, reflect.Array:
						b.bytes(w.formatSlice(at, av, 0)...)
					case reflect.Map:
						b.bytes(w.formatMap(at, av, 0)...)
					case reflect.Struct:
						b.bytes(w.formatStruct(at, av, 0)...)
					default:
						b.string(fmt.Sprint(value))
					}
				}
				b.bytes('\n')
			}
		}))
}

type event struct {
	ts     time.Time
	level  zerolog.Level
	fields map[string]any
}

func (e *event) Str(k string, v string) *event {
	e.fields[k] = v
	return e
}

func (e *event) Time(k string, v time.Time) *event {
	e.fields[k] = v
	return e
}

func (e *event) Any(k string, v any) *event {
	e.fields[k] = v
	return e
}

func (e *event) Err(err error) *event {
	e.fields[zerolog.ErrorFieldName] = err
	return e
}

func (e *event) Errs(k string, errs []error) *event {
	e.fields[k] = errs
	return e
}

func (e *event) Msg(msg string) {
	(&prettyWriter{
		maxSlicePrintSize: 0,
		b:                 newBuffer(os.Stderr),
	}).write(msg, e)
}

type logger struct{}

var _logger = &logger{}

func (l *logger) log(level zerolog.Level) *event {
	return &event{
		ts:     time.Now(),
		level:  level,
		fields: map[string]any{},
	}
}

func (l *logger) Info() *event {
	return l.log(zerolog.InfoLevel)
}

func (l *logger) Warn() *event {
	return l.log(zerolog.WarnLevel)
}

func (l *logger) Error() *event {
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

func Info() *event {
	return _logger.Info()
}

func Warn() *event {
	return _logger.Warn()
}

func Error() *event {
	return _logger.Error()
}

func Fatal(err error) {
	_logger.Fatal(err)
}
