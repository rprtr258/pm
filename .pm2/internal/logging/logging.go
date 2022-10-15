package logging

import (
	"fmt"
	"io"
	"os"

	"github.com/rprtr258/pm/internal/constants"
)

type Logger bool

func (l Logger) Err(msg string) {
	l.log(os.Stdout, constants.PREFIX_MSG_ERR, msg)
}

func (l Logger) Log(msg string) {
	l.log(os.Stdout, constants.PREFIX_MSG, msg)
}

func (l Logger) Info(msg string) {
	l.log(os.Stdout, constants.PREFIX_MSG_INFO, msg)
}

func (l Logger) Warn(msg string) {
	l.log(os.Stdout, constants.PREFIX_MSG_WARNING, msg)
}

func (l Logger) LogMod(msg string) {
	l.log(os.Stdout, constants.PREFIX_MSG_MOD, msg)
}

func (l Logger) ErrMod(msg string) {
	l.log(os.Stderr, constants.PREFIX_MSG_MOD_ERR, msg)
}

func (l Logger) log(out io.Writer, prefix, msg string) {
	if !l {
		return
	}
	fmt.Fprintf(out, "%s%s\n", prefix, msg)
}
