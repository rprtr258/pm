package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	log.Logger = zerolog.New(os.Stderr).With().
		Timestamp().
		Caller().
		Logger()

	if err := cli.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
