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
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/scuf"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog"
)

func colorStringFg(bb []byte, color []byte) []byte {
	return []byte(scuf.String(string(bb), color))
}

type prettyWriter struct {
	maxSlicePrintSize int
	b                 scuf.Buffer
}

func levelColors(level zerolog.Level) (bg, fg []byte) { //nolint:nonamedreturns // for documentation purposes
	switch {
	case level < zerolog.InfoLevel:
		return scuf.BgBlue, scuf.FgBlue
	case level < zerolog.WarnLevel:
		return scuf.BgGreen, scuf.FgGreen
	case level < zerolog.ErrorLevel:
		return scuf.BgYellow, scuf.FgYellow
	default:
		return scuf.BgRed, scuf.FgRed
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
	scuf.New(&bb).
		Bytes(w.buildTypeString(st.String())...).
		InBytePair('(', ')', func(b scuf.Buffer) {
			b.String(strconv.Itoa(sv.Len()), scuf.FgBlue)
		}).
		Iter(iter.Map(iter.FromRange(0, min(sv.Len(), w.maxSlicePrintSize+1), 1), func(i int) func(scuf.Buffer) {
			return func(b scuf.Buffer) {
				if i == w.maxSlicePrintSize {
					b.
						Bytes('\n').
						RepeatByte(' ', l*2+4).
						RepeatByte(' ', d+2).
						String("...", scuf.FgBlue).
						String("]", scuf.FgGreen)
					return
				}

				v := sv.Index(i)
				tb := strconv.Itoa(i)
				b.
					Bytes('\n').
					RepeatByte(' ', l*2+4).
					RepeatByte(' ', d-len(tb)).
					String(tb, scuf.FgGreen).
					Bytes(' ').
					Bytes(w.formatValue(v, l+1)...)
			}
		}))
	return bb.Bytes()
}

func (w *prettyWriter) formatMap(typ reflect.Type, val reflect.Value, l int) []byte {
	p := 0
	for _, k := range val.MapKeys() {
		p = max(p, len(anyToStr(k)))
	}
	p += len(scuf.FgGreen) + len(scuf.ModReset)

	sk := val.MapKeys()
	slices.SortFunc(sk, func(i, j reflect.Value) int {
		return cmp.Compare(anyToStr(i), anyToStr(j))
	})

	var bb bytes.Buffer
	scuf.New(&bb).
		Bytes(w.buildTypeString(typ.String())...).
		InBytePair('(', ')', func(b scuf.Buffer) {
			b.String(strconv.Itoa(val.Len()), scuf.FgBlue)
		}).
		Iter(iter.Map(iter.FromMany(sk...), func(k reflect.Value) func(scuf.Buffer) {
			return func(b scuf.Buffer) {
				tb := colorStringFg(w.formatValue(k, l+1), scuf.FgGreen)
				b.
					Bytes('\n').
					RepeatByte(' ', l*2+4).
					Bytes(tb...).
					RepeatByte(' ', max(0, p-len(tb))).
					Bytes(' ').
					Bytes(w.formatValue(val.MapIndex(k), l+1)...)
			}
		}))
	return bb.Bytes()
}

func (w *prettyWriter) formatStruct(st reflect.Type, sv reflect.Value, l int) []byte {
	p := 0
	for i := 0; i < st.NumField(); i++ {
		if sv.Type().Field(i).IsExported() {
			p = max(p, len(st.Field(i).Name))
		}
	}
	p += len(scuf.FgGreen) + len(scuf.ModReset)

	zeroes := 0
	var bb bytes.Buffer
	scuf.New(&bb).
		Bytes(w.buildTypeString(st.String())...).
		Iter(iter.Map(iter.FromRange(0, st.NumField(), 1), func(i int) func(scuf.Buffer) {
			return func(b scuf.Buffer) {
				if !sv.Type().Field(i).IsExported() {
					return
				}

				val := sv.Field(i)
				if val.IsZero() {
					zeroes++
					return
				}

				fieldName := colorStringFg([]byte(sv.Type().Field(i).Name), scuf.FgGreen)

				b.
					Bytes('\n').
					RepeatByte(' ', l*2+4).
					Bytes(fieldName...).
					RepeatByte(' ', max(0, p-len(fieldName))).
					Bytes(' ').
					Bytes(w.formatValue(val, l+1)...)
			}
		})).
		Styled(func(b scuf.Buffer) {
			if zeroes > 0 {
				b.
					Bytes('\n').
					RepeatByte(' ', l*2+4).
					String("// zeros", scuf.ModFaint)
			}
		})
	return bb.Bytes()
}

func (w *prettyWriter) formatValue(v reflect.Value, l int) []byte {
	if v.IsZero() {
		var bb bytes.Buffer
		scuf.New(&bb).
			String(anyToStr(v), scuf.ModFaint)
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
		res = []byte(anyToStr(v))
	}

	if s := anyToStr(v); s == "<nil>" || s == "0" || s == "false" {
		return []byte(scuf.NewString(func(b scuf.Buffer) {
			b.Styled(func(b scuf.Buffer) {
				b.Bytes(res...)
			}, scuf.FgWhite, scuf.ModFaint)
		}))
	}

	return res
}

func (w *prettyWriter) buildTypeString(typeStr string) []byte {
	typeStr = strings.ReplaceAll(typeStr, "interface {}", "any")
	var bb bytes.Buffer
	scuf.New(&bb).
		Iter(iter.Map(iter.FromMany([]byte(typeStr)...), func(c byte) func(scuf.Buffer) {
			return func(b scuf.Buffer) {
				switch c {
				case '*':
					b.Styled(func(b scuf.Buffer) {
						b.Bytes(c)
					}, scuf.FgRed)
				case '[', ']':
					b.Styled(func(b scuf.Buffer) {
						b.Bytes(c)
					}, scuf.FgGreen)
				default:
					b.Styled(func(b scuf.Buffer) {
						b.Bytes(c)
					}, scuf.FgYellow)
				}
			}
		}))
	return bb.Bytes()
}

// anyToStr using fmt.Sprint
func anyToStr(a reflect.Value) string {
	return fmt.Sprint(a.Interface())
}

func (w *prettyWriter) write(msg string, ev *Event) {
	colorBg, colorFg := levelColors(ev.level)

	padding := 0
	for k := range ev.fields {
		padding = max(padding, len(k))
	}
	padding += len(scuf.FgMagenta) + len(scuf.ModReset)

	w.b.
		String(ev.ts.Format("[15:06:05]"), scuf.ModFaint, scuf.FgWhite).
		Bytes(' ').
		// level
		Styled(func(b scuf.Buffer) {
			b.InBytePair(' ', ' ', func(b scuf.Buffer) {
				b.String(strings.ToUpper(ev.level.String()))
			})
		}, scuf.FgBlack, colorBg).
		Bytes(' ').
		String(msg, colorFg).
		Bytes('\n').
		// attributes
		Iter(iter.Map(iterSorted(iter.FromDict(ev.fields)), func(kv fun.Pair[string, any]) func(scuf.Buffer) {
			k, value := kv.K, kv.V
			return func(b scuf.Buffer) {
				b.
					String(k, scuf.FgMagenta).
					RepeatByte(' ', padding-len(k)).
					Bytes(' ')
				switch vv := value.(type) {
				case time.Time, time.Duration:
					b.String(fmt.Sprint(vv), scuf.FgCyan)
				case *time.Time:
					b.String(vv.String(), scuf.FgCyan)
				case *time.Duration:
					b.String(vv.String(), scuf.FgCyan)
				default:
					at := reflect.TypeOf(value)
					av := reflect.ValueOf(value)
					switch at.Kind() { //nolint:exhaustive // not needed
					case reflect.Float32, reflect.Float64,
						reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
						reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
						b.String(fmt.Sprint(value), scuf.FgYellow)
					case reflect.Bool:
						b.String(fmt.Sprint(value), scuf.FgRed)
					case reflect.String:
						v, ok := value.(string) //nolint:forcetypeassert,errcheck // checked kind already
						if !ok {
							v = value.(core.PMID).String()
						}
						switch {
						case v == "":
							b.String("empty", scuf.FgWhite, scuf.ModFaint)
						case isURL(v):
							b.String(v, scuf.FgBlue, scuf.ModUnderline)
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

func (e *Event) Stringer(k string, v fmt.Stringer) *Event {
	return e.Str(k, v.String())
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
		b:                 scuf.New(os.Stderr),
	}).write(msg, e)
}

func (e *Event) Send() {
	e.Msg("")
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

func (l *logger) Debug() *Event {
	return l.log(zerolog.DebugLevel)
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
	e = e.Str("errstr", err.Error())

	e.Msg("app exited abnormally")
	os.Exit(1)
}

func Debug() *Event {
	return _logger.Debug()
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
