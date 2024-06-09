package main

import (
	"os"
	"strings"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func newLogger() zerolog.Logger {
	return zerolog.New(os.Stderr).With().
		Timestamp().
		Logger().
		Output(zerolog.ConsoleWriter{ //nolint:exhaustruct // not needed
			Out: os.Stderr,
			FormatLevel: func(i interface{}) string {
				s, _ := i.(string)
				bg := fun.Switch(s, scuf.BgRed).
					Case(scuf.BgBlue, zerolog.LevelInfoValue).
					Case(scuf.BgGreen, zerolog.LevelWarnValue).
					Case(scuf.BgYellow, zerolog.LevelErrorValue).
					End()

				return scuf.String(" "+strings.ToUpper(s)+" ", bg, scuf.FgBlack)
			},
			FormatTimestamp: func(i interface{}) string {
				s, _ := i.(string)
				t, err := time.Parse(zerolog.TimeFieldFormat, s)
				if err != nil {
					return s
				}

				return scuf.String(t.Format("[15:06:05]"), scuf.ModFaint, scuf.FgWhite)
			},
		})
}

func main() {
	log.Logger = newLogger()

	if err := cli.Run(os.Args); err != nil {
		_ = err // NOTE: ignore, since cobra will print the error
	}
}
