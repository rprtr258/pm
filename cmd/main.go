package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/cmd/internal"
	internal2 "github.com/rprtr258/pm/internal"
)

func main() {
	app := &cli.App{
		Name:  "pm",
		Usage: "manage running processes",
		Flags: []cli.Flag{
			// If sets and script’s memory usage goes about the configured number, pm2 restarts the script.
			// Uses human-friendly suffixes: ‘K’ for kilobytes, ‘M’ for megabytes, ‘G’ for gigabytes’, etc. Eg “150M”.
			// &cli.IntFlag{Name: "max-memory-restart", Usage: "Restart the app if an amount of memory is exceeded (in bytes)"},
			// &cli.BoolFlag{Name:        "attach", Usage: "attach logging after your start/restart/stop/reload"},
			// &cli.DurationFlag{Name:    "listen-timeout", Usage: "listen timeout on application reload"},
			// &cli.BoolFlag{Name:        "no-color", Usage: "skip colors"},
			// &cli.BoolFlag{Name:        "no-daemon", Usage: "run pm2 daemon in the foreground if it doesn\t exist already"},
			// &cli.BoolFlag{Name:        "no-vizion", Usage: "start an app without vizion feature (versioning control)"},
			// &cli.IntFlag{Name:         "parallel", Usage: "number of parallel actions (for restart/reload)"},
			// &cli.BoolFlag{Name:        "silent", Aliases: []string{"s"}, Usage: "hide all messages", Value: false},
			// &cli.BoolFlag{Name:        "wait-ip",
			//               Usage: "override systemd script to wait for full internet connectivity to launch pm2"},
		},
		Commands: append(
			internal.AllCmds,
			&cli.Command{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "print pm version",
				// TODO: implement
			},
		),
		Before: func(*cli.Context) error {
			// TODO: run daemon if not running
			_, err := os.Stat(internal2.DirHome)
			if err != nil {
				if !os.IsNotExist(err) {
					return xerr.NewWM(err, "os.stat", xerr.Field("homedir", internal2.DirHome))
				}

				if err := os.Mkdir(internal2.DirHome, 0o755); err != nil {
					return xerr.NewWM(err, "create dir", xerr.Field("homedir", internal2.DirHome))
				}
			}

			return nil
		},
	}

	rand.Seed(time.Now().UnixNano())

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
