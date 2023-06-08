package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/cmd/internal"
	internal2 "github.com/rprtr258/pm/internal"
)

func ensureDir(dirname string) error {
	_, errStat := os.Stat(dirname)
	if errStat == nil {
		return nil
	}

	if !os.IsNotExist(errStat) {
		return xerr.NewWM(errStat, "stat home dir")
	}

	log.Infof("creating home dir...", log.F{"dir": dirname})
	if errMkdir := os.Mkdir(dirname, 0o755); errMkdir != nil {
		return xerr.NewWM(errMkdir, "create home dir")
	}

	return nil
}

var _app = &cli.App{
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

		if err := ensureDir(internal2.DirHome); err != nil {
			return xerr.NewWM(err, "ensure home dir", xerr.Fields{"dir": internal2.DirHome})
		}

		return nil
	},
}

func main() {
	if errRun := _app.Run(os.Args); errRun != nil {
		log.Fatal(errRun.Error())
	}
}
