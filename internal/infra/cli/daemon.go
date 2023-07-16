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
				pid, errRestart := pm_daemon.Restart(ctx.Context)
				if errRestart != nil {
					return xerr.NewWM(errRestart, "restart daemon process")
				}

				// if in client, print daemon pid
				if pid != 0 {
					fmt.Println(pid)
				}

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
				if errRun := pm_daemon.RunServer(ctx.Context); errRun != nil {
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
		{
			Name:  "logs",
			Usage: "check daemon logs",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "follow",
					Aliases: []string{"f"},
					Usage:   "follow logs",
					Value:   false,
				},
			},
			Action: func(ctx *cli.Context) error {
				follow := ctx.Bool("follow")

				if errLogs := pm_daemon.Logs(ctx.Context, follow); errLogs != nil {
					return xerr.NewWM(errLogs, "check daemon logs")
				}

				return nil
			},
		},
	},
}
