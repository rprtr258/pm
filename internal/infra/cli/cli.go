package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/cli/log/buffer"
)

func ensureDir(dirname string) error {
	_, errStat := os.Stat(dirname)
	if errStat == nil {
		return nil
	}
	if !errors.Is(errStat, fs.ErrNotExist) {
		return xerr.NewWM(errStat, "stat dir")
	}

	log.Info().Str("dir", dirname).Msg("creating home dir...")
	if errMkdir := os.Mkdir(dirname, 0o755); errMkdir != nil {
		return xerr.NewWM(errMkdir, "create dir")
	}

	return nil
}

//nolint:lll // setting template strings
func Init() {
	cli.AppHelpTemplate = buffer.NewString(func(b *buffer.Buffer) {
		b.
			String(`{{template "helpNameTemplate" .}}`, buffer.FgBlue).
			String(`

Usage:
	{{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.HelpName}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Description}}

Description:
   {{template "descriptionTemplate" .}}{{end}}
{{- if len .Authors}}

Author{{template "authorsTemplate" .}}{{end}}{{if .VisibleCommands}}

Commands:{{range .VisibleCategories}}{{if .Name}}
   `).
			String(`{{.Name}}`, buffer.FgCyan).
			String(`:{{range .VisibleCommands}}
     `).
			String(`{{join .Names ", "}}`, buffer.FgGreen).
			String(`{{"\t"}}`).
			String(`{{.Usage}}`, buffer.FgWhite).
			String(`{{end}}{{else}}{{ $cv := offsetCommands .VisibleCommands 5}}{{range .VisibleCommands}}
   {{$s := join .Names ", "}}`).
			String(`{{$s}}`, buffer.FgGreen).
			String(`{{ $sp := subtract $cv (offset $s 3) }}{{ indent $sp ""}}`).
			String(`{{wrap .Usage $cv}}`, buffer.FgWhite).
			String(`{{end}}{{end}}{{end}}{{end}}
`)
	})
	cli.CommandHelpTemplate = buffer.NewString(func(b *buffer.Buffer) {
		b.
			String(`{{template "helpNameTemplate" .}}`, buffer.FgBlue).
			String(`

Usage:
   {{template "usageTemplate" .}}{{if .Description}}

Description:
   {{template "descriptionTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

Options:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

Options:{{range $i, $e := .VisibleFlags}}
   `).
			// TODO: paint flags (before \t), dont paint description (after \t)
			String(`{{$e.String}}`, buffer.FgGreen).
			String(`{{end}}{{end}}
`) // TODO: color flags similar to coloring commands in app help
		// TODO: fix coloring for `pm ls --help“
	})
}

var App = &cli.App{
	Name:    "pm",
	Version: "0.2.0", // TODO: set at compile time
	Usage:   "manage running processes",
	Flags:   []cli.Flag{
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
		_runCmd, _startCmd, _restartCmd, _stopCmd, _deleteCmd,
		_listCmd, _logsCmd,
		_versionCmd,
	},
	HideHelpCommand: true,
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
