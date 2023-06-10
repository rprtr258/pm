package main

import (
	"os"

	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	if errRun := cli.App.Run(os.Args); errRun != nil {
		message, fields := xerr.UnwrapFields(errRun)
		log.Fatalf(message, fields)
	}
}
