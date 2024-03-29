package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rprtr258/xerr"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"

	"github.com/rprtr258/pm/internal/core"
)

func ensureDir(dirname string) error {
	_, errStat := os.Stat(dirname)
	if errStat == nil {
		return nil
	}
	if !errors.Is(errStat, fs.ErrNotExist) {
		return xerr.NewWM(errStat, "stat dir")
	}

	slog.Info("creating home dir...", "dir", dirname)
	if errMkdir := os.Mkdir(dirname, 0o755); errMkdir != nil {
		return xerr.NewWM(errMkdir, "create dir")
	}

	return nil
}

var App = &cli.App{
	Name:  "pm",
	Usage: "manage running processes",
	Flags: []cli.Flag{
		// If sets and script’s memory usage goes about the configured number, pm2 restarts the script.
		// Uses human-friendly suffixes: ‘K’ for kilobytes, ‘M’ for megabytes, ‘G’ for gigabytes’, etc. Eg “150M”.
		// &cli.IntFlag{Name: "max-memory-restart", Usage: "Restart the app if an amount of memory is exceeded (in bytes)"},
		// &cli.BoolFlag{Name:        "attach", Usage: "attach logging after your start/restart/stop/reload"},
		// &cli.DurationFlag{Name:    "listen-timeout", Usage: "listen timeout on application reload"},
		// &cli.BoolFlag{Name:        "no-daemon", Usage: "run pm2 daemon in the foreground if it doesn\t exist already"},
		// &cli.BoolFlag{Name:        "no-vizion", Usage: "start an app without vizion feature (versioning control)"},
		// &cli.IntFlag{Name:         "parallel", Usage: "number of parallel actions (for restart/reload)"},
		// &cli.BoolFlag{Name:        "silent", Aliases: []string{"s"}, Usage: "hide all messages", Value: false},
		// &cli.BoolFlag{Name:        "wait-ip",
		//               Usage: "override systemd script to wait for full internet connectivity to launch pm2"},
	},
	Commands: []*cli.Command{
		_daemonCmd,
		// process management
		_runCmd, _startCmd, _stopCmd, _deleteCmd,
		// inspection
		_listCmd,
		// other
		_versionCmd,
	},
	Before: func(c *cli.Context) error {
		if err := ensureDir(core.DirHome); err != nil {
			return xerr.NewWM(err, "ensure home dir", xerr.Fields{"dir": core.DirHome})
		}

		_dirProcsLogs := filepath.Join(core.DirHome, "logs")
		if err := ensureDir(_dirProcsLogs); err != nil {
			return xerr.NewWM(err, "ensure logs dir", xerr.Fields{"dir": _dirProcsLogs})
		}

		return nil
	},
}
