package main

import (
	"os"

	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		log.Fatal().Msg(err.Error())
	}
}
