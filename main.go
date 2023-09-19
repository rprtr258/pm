package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	log.Logger = zerolog.New(os.Stderr).With().
		Timestamp().
		Caller().
		Logger()
	// TODO: if not daemon
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	color.NoColor = false

	cli.Init()
	if errRun := cli.App.Run(os.Args); errRun != nil {
		log.Fatal().
			Func(func(e *zerolog.Event) {
				if err, ok := errRun.(*xerr.Error); ok {
					e.
						Str("msg", err.Message).
						Err(err.Err)
					if len(err.Errs) > 0 {
						e.Errs("errs", err.Errs)
					}
					if !err.At.IsZero() {
						e.Time("at", err.At)
					}
					if caller := err.Caller; caller != nil {
						e.Str("err_caller", fmt.Sprintf("%s:%d#%s", caller.File, caller.Line, caller.Function))
					}
					for i, frame := range err.Stacktrace {
						e.Str(fmt.Sprintf("stack[%d]", i), fmt.Sprintf("%s:%d#%s", frame.File, frame.Line, frame.Function))
					}
					for k, v := range err.Fields {
						e.Any(k, v)
					}
				} else {
					e.Err(errRun)
				}
			}).
			Msg("app exited abnormally")
	}
}
