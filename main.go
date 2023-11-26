package main

import (
	"os"

	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/cli"
	"github.com/rprtr258/pm/internal/infra/log"
)

func main() {
	zerologlog.Logger = zerolog.New(os.Stderr).With().
		Timestamp().
		Caller().
		Logger()

	cli.Init()
	if errRun := cli.App.Run(os.Args); errRun != nil {
		log.Fatal(errRun)
	}
}
