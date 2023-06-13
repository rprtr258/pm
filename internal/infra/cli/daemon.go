package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	pm_daemon "github.com/rprtr258/pm/internal/core/daemon"
)

var _daemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "manage daemon",
	Subcommands: []*cli.Command{
		{
			Name:    "start",
			Aliases: []string{"restart"},
			Usage:   "launch daemon process",
			Action: func(ctx *cli.Context) error {
				pid, errRestart := pm_daemon.Restart()
				if errRestart != nil {
					return xerr.NewWM(errRestart, "restart daemon process")
				}

				fmt.Println(pid)

				return nil
			},
		},
		{
			Name:    "stop",
			Aliases: []string{"kill"},
			Usage:   "stop daemon process",
			Action: func(ctx *cli.Context) error {
				if errStop := pm_daemon.Kill(); errStop != nil {
					return xerr.NewWM(errStop, "stop daemon process")
				}

				return nil
			},
		},
		{
			Name:  "run",
			Usage: "run daemon server without daemonizing, DON'T USE BY HAND IF YOU DON'T KNOW WHAT YOU ARE DOING",
			Action: func(ctx *cli.Context) error {
				if errRun := pm_daemon.RunServer(); errRun != nil {
					return xerr.NewWM(errRun, "run daemon process")
				}

				return nil
			},
		},
		{
			Name:    "status",
			Usage:   "check daemon status",
			Aliases: []string{"ps"},
			Action: func(ctx *cli.Context) error {
				if errStatus := pm_daemon.Status(ctx.Context); errStatus != nil {
					return xerr.NewWM(errStatus, "check daemon status")
				}

				fmt.Println("ok")

				return nil
			},
		},
	},
}
