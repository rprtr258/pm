package main

import (
	"os"

	flags "github.com/rprtr258/cli/contrib"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	zerologlog.Logger = zerolog.New(os.Stderr).With().
		Timestamp().
		Caller().
		Logger()

	if _, err := cli.Parser.ParseArgs(os.Args[1:]...); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Kind == flags.ErrHelp {
			return
		}
		cli.Parser.WriteHelp(os.Stderr)
		// fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
