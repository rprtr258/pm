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

func errFunc(e *zerolog.Event, prefix string, err_orig error) {
	if err, ok := err_orig.(*xerr.Error); ok {
		e.Str("msg", err.Message)
		if err.Err != nil {
			d := zerolog.Dict()
			errFunc(d, prefix+".err", err.Err)
			e.Dict(prefix+"err", d)
		}
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
		e.AnErr(prefix+"err", err_orig)
	}
}

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
				errFunc(e, "", errRun)
			}).
			Msg("app exited abnormally")
	}
}
