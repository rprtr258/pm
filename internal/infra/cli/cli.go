package cli

import (
	"context"
	stdErrors "errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/rprtr258/cli"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/infra/errors"
)

func ensureDir(dirname string) error {
	_, errStat := os.Stat(dirname)
	if errStat == nil {
		return nil
	}
	if !stdErrors.Is(errStat, fs.ErrNotExist) {
		return errors.Wrap(errStat, "stat dir")
	}

	log.Info().Str("dir", dirname).Msg("creating dir...")
	if errMkdir := os.Mkdir(dirname, 0o755); errMkdir != nil {
		return errors.Wrap(errMkdir, "create dir")
	}

	return nil
}

type App struct {
	// 	Name:    "pm",
	// 	Usage:   "manage running processes",

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
	// App struct {
	Version _cmdVersion `command:"version" description:"print pm version"`
	Agent   _cmdAgent   `command:"agent" hidden:"yes"`
	// } `category:""` // TODO: unused
	// Inspection struct {
	List    _cmdList    `command:"list" description:"list processes" alias:"l" alias:"ls" alias:"ps" alias:"status"`
	Logs    _cmdLogs    `command:"logs" description:"watch for processes logs"`
	Inspect _cmdInspect `command:"inspect" description:"inspect process" alias:"i"`
	// } `category:"inspection"` // TODO: unused
	// Management struct {
	Run     _cmdRun     `command:"run" description:"create and run new process"`
	Start   _cmdStart   `command:"start" description:"start already added process(es)"`
	Restart _cmdRestart `command:"restart" description:"restart already added process(es)"`
	Stop    _cmdStop    `command:"stop" description:"stop process(es)" alias:"kill"`
	Delete  _cmdDelete  `command:"delete" description:"stop and remove process(es)" alias:"del" alias:"rm"`
	Signal  _cmdSignal  `command:"signal" description:"send signal to process(es)"`
	// } `category:"management"`
}

func Run(argv []string) error {
	// 	Before: func(c *flags.Context) error {
	// 		if err := ensureDir(core.DirHome); err != nil {
	// 			return errors.Wrap(err, "ensure home dir", map[string]any{"dir": core.DirHome})
	// 		}

	// 		_dirProcsLogs := filepath.Join(core.DirHome, "logs")
	// 		if err := ensureDir(_dirProcsLogs); err != nil {
	// 			return errors.Wrap(err, "ensure logs dir", map[string]any{"dir": _dirProcsLogs})
	// 		}

	// 		return nil
	// 	},

	//nolint:lll // setting template strings
	// func Init() {
	// 	flags.AppHelpTemplate = scuf.NewString(func(b scuf.Buffer) {
	// 		b.
	// 			String(`{{template "helpNameTemplate" .}}`, scuf.FgBlue).
	// 			String(`

	// Usage:
	// 	{{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.HelpName}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Description}}

	// Description:
	//    {{template "descriptionTemplate" .}}{{end}}
	// {{- if len .Authors}}

	// Author{{template "authorsTemplate" .}}{{end}}{{if .VisibleCommands}}

	// Commands:{{range .VisibleCategories}}{{if .Name}}
	//    `).
	// 			String(`{{.Name}}`, scuf.FgCyan).
	// 			String(`:{{range .VisibleCommands}}
	//      `).
	// 			String(`{{join .Names ", "}}`, scuf.FgGreen).
	// 			String(`{{"\t"}}`).
	// 			String(`{{.Usage}}`, scuf.FgWhite).
	// 			String(`{{end}}{{else}}{{ $cv := offsetCommands .VisibleCommands 5}}{{range .VisibleCommands}}
	//    {{$s := join .Names ", "}}`).
	// 			String(`{{$s}}`, scuf.FgGreen).
	// 			String(`{{ $sp := subtract $cv (offset $s 3) }}{{ indent $sp ""}}`).
	// 			String(`{{wrap .Usage $cv}}`, scuf.FgWhite).
	// 			String(`{{end}}{{end}}{{end}}{{end}}
	// `)
	// 	})
	// 	flags.CommandHelpTemplate = scuf.NewString(func(b scuf.Buffer) {
	// 		b.
	// 			String(`{{template "helpNameTemplate" .}}`, scuf.FgBlue).
	// 			String(`

	// Usage:
	//    {{template "usageTemplate" .}}{{if .Description}}

	// Description:
	//    {{template "descriptionTemplate" .}}{{end}}{{if .VisibleFlagCategories}}

	// Options:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

	// Options:{{range $i, $e := .VisibleFlags}}
	//    `).
	// 			// TODO: paint flags (before \t), dont paint description (after \t)
	// 			String(`{{$e.String}}`, scuf.FgGreen).
	// 			String(`{{end}}{{end}}
	// `) // TODO: color flags similar to coloring commands in app help
	// 		// TODO: fix coloring for `pm ls --help“
	// 	})
	// }

	if err := cli.RunContext[App](context.Background(), argv...); err != nil {
		if err, ok := err.(*cli.Error); ok && err.Kind == cli.ErrHelp {
			fmt.Println(err.Error())
			return nil
		}

		return err
	}

	return nil
}
