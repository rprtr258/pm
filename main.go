package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/rprtr258/log"
	"golang.org/x/exp/slog"

	"github.com/rprtr258/pm/internal/infra/cli"
)

func main() {
	slog.SetDefault(slog.New(log.NewDestructorHandler(log.NewPrettyHandler(os.Stderr))))

	color.NoColor = false
	cli.Init()
	if errRun := cli.App.Run(os.Args); errRun != nil {
		slog.Error("app exited abnormally", "err", errRun)
		os.Exit(1)
	}
}
