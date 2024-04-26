package cli

import (
	stdErrors "errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/scuf"
)

func ensureDir(dirname string) error {
	if _, errStat := os.Stat(dirname); errStat == nil {
		return nil
	} else if !stdErrors.Is(errStat, fs.ErrNotExist) {
		return errors.Wrapf(errStat, "stat dir")
	}

	log.Info().Str("dir", dirname).Msg("creating dir...")
	if errMkdir := os.Mkdir(dirname, 0o755); errMkdir != nil {
		return errors.Wrapf(errMkdir, "create dir")
	}

	return nil
}

var App = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "pm",
		Short:        "manage running processes",
		SilenceUsage: true,
	}
	// If sets and script’s memory usage goes about the configured number, pm2 restarts the script.
	// Uses human-friendly suffixes: ‘K’ for kilobytes, ‘M’ for megabytes, ‘G’ for gigabytes’, etc. Eg “150M”.
	// &cli.IntFlag{Name: "max-memory-restart", Usage: "Restart the app if an amount of memory is exceeded (in bytes)"},
	// &cli.BoolFlag{Name:        "attach", Usage: "attach logging after your start/restart/stop/reload"},
	// &cli.DurationFlag{Name:    "listen-timeout", Usage: "listen timeout on application reload"},
	// &cli.BoolFlag{Name:        "no-daemon", Usage: "run pm2 daemon in the foreground if it doesn\t exist already"},
	// &cli.BoolFlag{Name:        "no-vizion", Usage: "start an app without vizion feature (versioning control)"},
	// &cli.IntFlag{Name:         "parallel", Usage: "number of parallel actions (for restart/reload)"},
	// &cli.BoolFlag{Name:        "silent", Aliases: []string{"s"}, Usage: "hide all messages", Value: false},
	// &cli.BoolFlag{Name:        "wait-ip", Usage: "override systemd script to wait for full internet connectivity to launch pm2"},

	cmd.AddCommand(_cmdVersion)
	cmd.AddCommand(_cmdAgent)
	cmd.AddCommand(_cmdStartup)

	cmd.AddGroup(&cobra.Group{ID: "inspection", Title: "Inspection:"})
	cmd.AddCommand(_cmdList)
	cmd.AddCommand(_cmdLogs)
	cmd.AddCommand(_cmdInspect)

	cmd.AddGroup(&cobra.Group{ID: "management", Title: "Management:"})
	cmd.AddCommand(_cmdRun)
	cmd.AddCommand(_cmdStart)
	cmd.AddCommand(_cmdRestart)
	cmd.AddCommand(_cmdStop)
	cmd.AddCommand(_cmdDelete)
	cmd.AddCommand(_cmdSignal)
	return cmd
}()

func Run(argv []string) error {
	if err := ensureDir(core.DirHome); err != nil {
		return errors.Wrapf(err, "ensure home dir %s", core.DirHome)
	}

	_dirProcsLogs := filepath.Join(core.DirHome, "logs")
	if err := ensureDir(_dirProcsLogs); err != nil {
		return errors.Wrapf(err, "ensure logs dir %s", _dirProcsLogs)
	}

	//nolint:lll // setting template strings
	// 	App.SetUsageFunc(func(cmd *cobra.Command) error {
	// 		scuf.New(os.Stdout).
	// 			NL().
	// 			String(`Usage:
	//   pm COMMAND
	// `).
	// 			Iter(func(yield func(func(scuf.Buffer)) bool) bool {
	// 				for _, group := range cmd.Groups() {
	// 					yield(func(b scuf.Buffer) {
	// 						b.NL().String(group.Title).NL()
	// 						for _, cmd := range cmd.Commands() {
	// 							if cmd.GroupID != group.ID {
	// 								continue
	// 							}
	// 							b.
	// 								String("  ").
	// 								String(cmd.Name(), scuf.FgCyan).
	// 								// String(strings.Join(cmd.Aliases, ", "), scuf.FgGreen).
	// 								String(strings.Repeat(" ", 12-len(cmd.Name()))).
	// 								String(cmd.Short).
	// 								NL()
	// 						}
	// 					})
	// 				}
	// 				return true
	// 			}).
	// 			String(`
	// Additional Commands:
	//   completion  Generate the autocompletion script for the specified shell
	//   help        Help about any command
	//   version     print pm version

	// Flags:
	//   -h, --help   help for pm

	// Use "pm [command] --help" for more information about a command.`)
	// 		return nil
	// 	})
	App.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		scuf.New(os.Stdout).
			String(cmd.Short, scuf.FgBlue).
			String(fmt.Sprintf(`

%s`,
				cmd.UsageString()))
	})

	App.SetArgs(argv[1:]) // TODO: govno ebanoe
	return App.Execute()
}
