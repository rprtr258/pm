package main

import (
	"os"

	"github.com/rprtr258/pm/internal/infra/cli"
	"github.com/rprtr258/pm/internal/infra/cli/log"
)

func main() {
	cli.Init()
	if errRun := cli.App.Run(os.Args); errRun != nil {
		log.Fatal(errRun)
	}
}
