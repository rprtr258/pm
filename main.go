package main

import (
	"os"

	"github.com/fatih/color"
	"golang.org/x/exp/slog"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	color.NoColor = false
	cli.Init()
	if errRun := cli.App.Run(os.Args); errRun != nil {
		slog.Error("app exited abnormally", "err", errRun)
		os.Exit(1)
	}
}
