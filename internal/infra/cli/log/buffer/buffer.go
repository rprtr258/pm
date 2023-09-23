package buffer

import (
	"bytes"
	"io"

	"github.com/rprtr258/fun/iter"
)

var (
	// Foreground colors
	FgBlack   = []byte("\x1b[30m")
	FgRed     = []byte("\x1b[31m")
	FgGreen   = []byte("\x1b[32m")
	FgYellow  = []byte("\x1b[33m")
	FgBlue    = []byte("\x1b[34m")
	FgMagenta = []byte("\x1b[35m")
	FgCyan    = []byte("\x1b[36m")
	FgWhite   = []byte("\x1b[37m")

	// Background colors
	BgBlack   = []byte("\x1b[40m")
	BgRed     = []byte("\x1b[41m")
	BgGreen   = []byte("\x1b[42m")
	BgYellow  = []byte("\x1b[43m")
	BgBlue    = []byte("\x1b[44m")
	BgMagenta = []byte("\x1b[45m")
	BgCyan    = []byte("\x1b[46m")
	BgWhite   = []byte("\x1b[47m")

	// Common consts
	ColorReset     = []byte("\x1b[0m")
	ColorFaint     = []byte("\x1b[2m")
	ColorUnderline = []byte("\x1b[4m")
)

type Buffer struct {
	out io.Writer
}

func New(out io.Writer) *Buffer {
	return &Buffer{out}
}

func (b *Buffer) Bytes(bs ...byte) *Buffer {
	b.out.Write(bs) //nolint:errcheck // fuck you
	return b
}

func (b *Buffer) RepeatByte(c byte, n int) *Buffer { //nolint:unparam // fuck you
	b.out.Write(bytes.Repeat([]byte{c}, n)) //nolint:errcheck // fuck you
	return b
}

func (b *Buffer) String(s string) *Buffer { //nolint:unparam // fuck off
	io.WriteString(b.out, s) //nolint:errcheck // fuck you
	return b
}

func (b *Buffer) Styled(f func(*Buffer), mods ...[]byte) *Buffer {
	for _, mod := range mods {
		b.out.Write(mod) //nolint:errcheck // fuck you
	}
	f(b)
	b.out.Write(ColorReset) //nolint:errcheck // fuck you
	return b
}

func (b *Buffer) InBytePair(start, end byte, f func(*Buffer)) *Buffer {
	return b.Styled(func(b *Buffer) {
		b.Bytes(start)
		f(b)
		b.Bytes(end)
	})
}

func (b *Buffer) Iter(seq iter.Seq[func(*Buffer)]) *Buffer {
	seq(func(f func(*Buffer)) bool {
		f(b)
		return true
	})
	return b
}
