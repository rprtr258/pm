package main

import (
	"os"

	"github.com/rprtr258/log"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	if errRun := cli.App.Run(os.Args); errRun != nil {
		log.Fatalf("app ended abnormally", log.F{"err": errRun})
	}
}
