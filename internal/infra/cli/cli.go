package cli

import (
	"errors"
	"io/fs"
	"os"

	flags "github.com/rprtr258/cli/contrib"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/infra/log"
)

func ensureDir(dirname string) error {
	_, errStat := os.Stat(dirname)
	if errStat == nil {
		return nil
	}
	if !errors.Is(errStat, fs.ErrNotExist) {
		return xerr.NewWM(errStat, "stat dir")
	}

	log.Info().Str("dir", dirname).Msg("creating dir...")
	if errMkdir := os.Mkdir(dirname, 0o755); errMkdir != nil {
		return xerr.NewWM(errMkdir, "create dir")
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
	App struct {
		Version _cmdVersion `command:"version" description:"print pm version"`
		Agent   _cmdAgent   `command:"agent" hidden:"yes"`
	} `category:""` // TODO: unused
	Inspection struct {
		List _cmdList `command:"list" description:"list processes" alias:"l" alias:"ls" alias:"ps" alias:"status"`
		Logs _cmdLogs `command:"logs" description:"watch for processes logs"`
	} `category:"inspection"` // TODO: unused
	Management struct {
		Run     _cmdRun     `command:"run" description:"create and run new process"`
		Start   _cmdStart   `command:"start" description:"start already added process(es)"`
		Restart _cmdRestart `command:"restart" description:"restart already added process(es)"`
		Stop    _cmdStop    `command:"stop" description:"stop process(es)"`
		Delete  _cmdDelete  `command:"delete" description:"stop and remove process(es)" alias:"del" alias:"rm"`
	} `category:"management"`
}

var Parser = func() *flags.Parser {
	parser := flags.NewParser(&App{}, flags.Default)
	for _, cmd := range parser.Commands() {
		if cmd.Name == "list" {
			cmd.FindOptionByLongName("format").Description = (*flagListFormat)(nil).Usage() // TODO: use as flag type method
			cmd.FindOptionByLongName("sort").Description = (*flagListSort)(nil).Usage()     // TODO: use as flag type method
		}
	}

	// 	Before: func(c *flags.Context) error {
	// 		if err := ensureDir(core.DirHome); err != nil {
	// 			return xerr.NewWM(err, "ensure home dir", xerr.Fields{"dir": core.DirHome})
	// 		}

	// 		_dirProcsLogs := filepath.Join(core.DirHome, "logs")
	// 		if err := ensureDir(_dirProcsLogs); err != nil {
	// 			return xerr.NewWM(err, "ensure logs dir", xerr.Fields{"dir": _dirProcsLogs})
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

	return parser
}()
