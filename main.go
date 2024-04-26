package main

import (
	"os"

	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/cli"
	mylog "github.com/rprtr258/pm/internal/infra/log"
)

func main() {
	log.Logger = mylog.New()

	if err := cli.Run(os.Args); err != nil {
		_ = err // NOTE: ignore, since cobra will print the error
	}
}
