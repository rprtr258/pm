package log

import (
	"os"
	"strings"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/rs/zerolog"
)

func New() zerolog.Logger {
	return zerolog.New(os.Stderr).With().
		Timestamp().
		Caller().
		Logger().
		Output(zerolog.ConsoleWriter{
			Out: os.Stderr,
			FormatLevel: func(i interface{}) string {
				s := i.(string)
				bg := fun.Switch(s, scuf.BgRed).
					Case(scuf.BgBlue, zerolog.LevelInfoValue).
					Case(scuf.BgGreen, zerolog.LevelWarnValue).
					Case(scuf.BgYellow, zerolog.LevelErrorValue).
					End()

				return scuf.String(" "+strings.ToUpper(s)+" ", bg, scuf.FgBlack)
			},
			FormatTimestamp: func(i interface{}) string {
				s := i.(string)
				t, err := time.Parse(zerolog.TimeFieldFormat, s)
				if err != nil {
					return s
				}

				return scuf.String(t.Format("[15:06:05]"), scuf.ModFaint, scuf.FgWhite)
			},
		})
}
