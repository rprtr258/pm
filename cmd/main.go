package main

import (
	"fmt"
	"os"
	"path"

	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/cmd/internal"
)

func main() {
	listCmd := &cli.Command{
		Name:    "list",
		Aliases: []string{"l"},
		Action: func(*cli.Context) error {
			fs, err := os.ReadDir(internal.HomeDir)
			if err != nil {
				return err
			}

			for _, f := range fs {
				if !f.IsDir() {
					fmt.Fprintf(os.Stderr, "found strange file %q which should not exist\n", path.Join(internal.HomeDir, f.Name()))
					continue
				}

				fmt.Printf("%#v", f.Name())
			}
			return nil
		},
	}

	stopCmd := &cli.Command{
		Name:      "stop",
		Usage:     "stop a process",
		ArgsUsage: "<id|name|namespace|all|json|stdin...>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "watch",
				Usage: "Stop watching folder for changes",
			},
		},
		Action: func(*cli.Context) error {
			//   .action(function(param) {
			//     forEachLimit(param, 1, function(script, next) {
			//       pm2.stop(script, next);
			//     }, function(err) {
			//       pm2.speedList(err ? 1 : 0);
			//     });
			return nil
		},
	}

	app := &cli.App{
		Name:  "pm",
		Usage: "manage running processes",
		Flags: []cli.Flag{
			// &cli.BoolFlag{Name: "version", Aliases: []string{"v"}, Usage: "print pm2 version"}, // version(pkg.version) // &cli.BoolFlag{Name: "silent", Aliases: []string{"s"}, Usage: "hide all messages", Value: false}, // &cli.StringSliceFlag{Name: "ext", Usage: "watch only this file extensions"}, // &cli.BoolFlag{Name: "mini-list", Aliases: []string{"m"}, Usage: "display a compacted list without formatting"}, // &cli.StringFlag{Name: "interpreter", Usage: "set a specific interpreter to use for executing app", Value: "node"}, // &cli.StringSliceFlag{Name: "interpreter-args", Usage: "set arguments to pass to the interpreter"}, // &cli.BoolFlag{Name: "output", Aliases: []string{"o"}, Usage: "specify log file for stdout"}, // &cli.PathFlag{Name: "error", Aliases: []string{"e"}, Usage: "specify log file for stderr"}, // &cli.PathFlag{Name: "log", Aliases: []string{"l"}, Usage: "specify log file which gathers both stdout and stderr"}, // &cli.StringSliceFlag{Name: "filter-env", Usage: "filter out outgoing global values that contain provided strings"}, // &cli.StringFlag{Name: "log-type", Usage: "specify log output style (raw by default, json optional)", Value: "raw"}, // &cli.StringFlag{Name: "log-date-format", Usage: "add custom prefix timestamp to logs"}, // &cli.BoolFlag{Name: "time", Usage: "enable time logging"}, // &cli.BoolFlag{Name: "disable-logs", Usage: "disable all logs storage"}, // &cli.StringFlag{Name: "env", Usage: "specify which set of environment variables from ecosystem file must be injected"}, // &cli.BoolFlag{Name: "update-env", Aliases: []string{"a"}, Usage: "force an update of the environment with restart/reload (-a <=> apply)"}, // &cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "force actions"}, // &cli.BoolFlag{Name: "instances <number>", Aliases: []string{"i"}, Usage: "launch [number] instances (for networked app)(load balanced)"}, // &cli.IntFlag{Name: "parallel", Usage: "number of parallel actions (for restart/reload)"}, // &cli.BoolFlag{Name: "shutdown-with-message", Usage: "shutdown an application with process.send('shutdown') instead of process.kill(pid, SIGINT)"}, // &cli.IntFlag{Name: "pid", Aliases: []string{"p"}, Usage: "specify pid file"}, // &cli.DurationFlag{Name: "kill-timeout", Aliases: []string{"k"}, Usage: "delay before sending final SIGKILL signal to process"}, // &cli.DurationFlag{Name: "listen-timeout", Usage: "listen timeout on application reload"}, // If sets and script’s memory usage goes about the configured number, pm2 restarts the script. Uses human-friendly suffixes: ‘K’ for kilobytes, ‘M’ for megabytes, ‘G’ for gigabytes’, etc. Eg “150M”.  // &cli.IntFlag{Name: "max-memory-restart", Usage: "Restart the app if an amount of memory is exceeded (in bytes)"}, // &cli.DurationFlag{Name: "restart-delay", Usage: "specify a delay between restarts"}, // &cli.DurationFlag{Name: "exp-backoff-restart-delay", Usage: "specify a delay between restarts"}, // &cli.BoolFlag{Name: "execute-command", Aliases: []string{"x"}, Usage: "execute a program using fork system"}, // &cli.IntFlag{Name: "max-restarts", Usage: "only restart the script COUNT times"}, // &cli.StringFlag{Name: "user", Aliases: []string{"u"}, Usage: "define user when generating startup script"}, // &cli.IntFlag{Name: "uid", Usage: "run target script with <uid> rights"}, // &cli.IntFlag{Name: "gid", Usage: "run target script with <gid> rights"}, // &cli.StringFlag{Name: "namespace", Usage: "start application within specified namespace"}, // &cli.StringFlag{Name: "cwd", Usage: "run target script from path <cwd>"}, // &cli.StringFlag{Name: "home-path", Usage: "define home path when generating startup script"}, // &cli.BoolFlag{Name: "wait-ip", Usage: "override systemd script to wait for full internet connectivity to launch pm2"}, // &cli.BoolFlag{Name: "service-name", Usage: "define service name when generating startup script"}, // &cli.StringFlag{Name: "cron-restart", Aliases: []string{"c", "cron"}, Usage: "restart a running process based on a cron pattern"}, // &cli.BoolFlag{Name: "write", Aliases: []string{"w"}, Usage: "write configuration in local folder"}, // &cli.BoolFlag{Name: "no-daemon", Usage: "run pm2 daemon in the foreground if it doesn\t exist already"}, // &cli.BoolFlag{Name: "source-map-support", Usage: "force source map support"}, // &cli.BoolFlag{Name: "only", Usage: "with json declaration, allow to only act on one application"}, // &cli.BoolFlag{Name: "disable-source-map-support", Usage: "force source map support"}, // &cli.BoolFlag{Name: "wait-ready", Usage: "ask pm2 to wait for ready event from your app"}, // &cli.BoolFlag{Name: "merge-logs", Usage: "merge logs from different instances but keep error and out separated"}, // &cli.StringSliceFlag{Name: "watch", Usage: "watch application folder for changes"}, // &cli.StringSliceFlag{Name: "ignore-watch", Usage: "List of paths to ignore (name or regex)"}, // &cli.DurationFlag{Name: "watch-delay", Usage: "specify a restart delay after changing files (--watch-delay 4 (in sec) or 4000ms)"}, // &cli.BoolFlag{Name: "no-color", Usage: "skip colors"}, // &cli.BoolFlag{Name: "no-vizion", Usage: "start an app without vizion feature (versioning control)"}, // &cli.BoolFlag{Name: "no-autorestart", Usage: "start an app without automatic restart"}, // &cli.BoolFlag{Name: "no-treekill", Usage: "Only kill the main process, not detached children"}, // &cli.BoolFlag{Name: "no-pmx", Usage: "start an app without pmx"}, // &cli.BoolFlag{Name: "no-automation", Usage: "start an app without pmx"}, // &cli.BoolFlag{Name: "trace", Usage: "enable transaction tracing with km"}, // &cli.BoolFlag{Name: "disable-trace", Usage: "disable transaction tracing with km"}, // &cli.BoolFlag{Name: "sort", Usage: "sort <field_name:sort> sort process according to field's name"}, // &cli.BoolFlag{Name: "attach", Usage: "attach logging after your start/restart/stop/reload"}, // &cli.BoolFlag{Name: "v8", Usage: "enable v8 data collecting"}, // &cli.BoolFlag{Name: "event-loop-inspector", Usage: "enable event-loop-inspector dump in pmx"}, // &cli.BoolFlag{Name: "deep-monitoring", Usage: "enable all monitoring tools (equivalent to --v8 --event-loop-inspector --trace)"},
		},
		Commands: []*cli.Command{
			internal.StartCmd,
			stopCmd,
			listCmd,
		},
		Before: func(*cli.Context) error {
			if _, err := os.Stat(internal.HomeDir); os.IsNotExist(err) {
				os.Mkdir(internal.HomeDir, 0755)
			}

			return nil
		},
	}

	//   pm2.getVersion(function(err, remote_version) { //     if (!err && (pkg.version != remote_version)) { //       console.log(''); //       console.log(chalk.red.bold('>>>> In-memory PM2 is out-of-date, do:\n>>>> $ pm2 update')); //       console.log('In memory PM2 version:', chalk.blue.bold(remote_version)); //       console.log('Local PM2 version:', chalk.blue.bold(pkg.version)); //       console.log(''); //     } //   });

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
