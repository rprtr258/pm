package main

import (
	"os"

	"github.com/rprtr258/log"
	"golang.org/x/exp/slog"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	slog.SetDefault(slog.New(log.New()))

	if errRun := cli.App.Run(os.Args); errRun != nil {
		slog.Error("app exited abnormally", "err", errRun)
		os.Exit(1)
	}
}
