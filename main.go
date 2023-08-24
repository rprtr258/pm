package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	color.NoColor = false
	cli.Init()
	if errRun := cli.App.Run(os.Args); errRun != nil {
		log.Fatal().
			Err(errRun).
			Msg("app exited abnormally")
	}
}
