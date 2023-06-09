package main

import (
	"os"

	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	if errRun := cli.App.Run(os.Args); errRun != nil {
		if s, e := xerr.MarshalJSON(errRun); e == nil {
			log.Fatal(string(s)) // TODO: get error fields
		} else {
			log.Fatal(errRun.Error())
		}
	}
}
