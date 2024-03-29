package cli

import (
	"fmt"
	"os"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/pkg/client"
)

var _runCmd = &cli.Command{
	Name:      "run",
	ArgsUsage: "cmd args...",
	Usage:     "run new process and manage it",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Usage:    "set a name for the process",
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:    "tag",
			Aliases: []string{"t"},
			Usage:   "add specified tag",
		},
		&cli.StringFlag{
			Name:  "cwd",
			Usage: "set working directory",
		},
		configFlag,
		// TODO: not yet implemented run parameters
		// // watch parameters
		// &cli.BoolFlag{Name: "watch-cwd", Usage: "Watch current working directory for changes"}, // or
		// &cli.StringSliceFlag{Name: "watch", Usage: "watch provided paths for changes"},
		// &cli.DurationFlag{
		// 	Name:  "watch-delay",
		// 	Usage: "specify a restart delay after changing files (--watch-delay 4 (in sec) or 4000ms)",
		// },
		// &cli.StringSliceFlag{Name: "ignore-watch", Usage: "List of paths to ignore (name or regex)"},
		// &cli.StringSliceFlag{Name: "ext", Usage: "watch files with these extensions"},
		// // logs parameters
		// &cli.BoolFlag{Name: "output", Aliases: []string{"o"}, Usage: "specify log file for stdout"},
		// &cli.PathFlag{Name: "error", Aliases: []string{"e"}, Usage: "specify log file for stderr"},
		// &cli.PathFlag{
		// 	Name:    "log",
		// 	Aliases: []string{"l"},
		// 	Usage:   "specify log file which gathers both stdout and stderr",
		// },
		// &cli.BoolFlag{Name: "disable-logs", Usage: "disable all logs storage"},
		// &cli.BoolFlag{Name: "time", Usage: "enable time logging"},
		// &cli.StringFlag{
		// 	Name:  "log-type",
		// 	Usage: "specify log output style (raw by default, json optional)",
		// 	Value: "raw",
		// },
		// // restart parameters
		// &cli.StringFlag{
		// 	Name:    "cron-restart",
		// 	Aliases: []string{"c", "cron"},
		// 	Usage:   "restart a running process based on a cron pattern",
		// },
		// &cli.IntFlag{
		// 	Name:  "max-restarts",
		// 	Usage: "only restart the script COUNT times",
		// },
		// &cli.BoolFlag{Name: "no-autorestart", Usage: "start an app without automatic restart"},
		// &cli.DurationFlag{Name: "restart-delay", Usage: "specify a delay between restarts"},
		// &cli.DurationFlag{Name: "exp-backoff-restart-delay", Usage: "specify a delay between restarts"},
		// // env parameters
		// &cli.StringFlag{
		// 	Name:  "env",
		// 	Usage: "specify which set of environment variables from ecosystem file must be injected",
		// },
		// &cli.StringSliceFlag{
		// 	Name:  "filter-env",
		// 	Usage: "filter out outgoing global values that contain provided strings",
		// },
		// // others
		// &cli.BoolFlag{Name: "wait-ready", Usage: "ask pm to wait for ready event from your app"},
		// &cli.IntFlag{
		// 	Name:    "pid",
		// 	Aliases: []string{"p"},
		// 	Usage:   "specify pid file",
		// },
		// &cli.IntFlag{Name: "gid", Usage: "run process with <gid> rights"},
		// &cli.IntFlag{Name: "uid", Usage: "run process with <uid> rights"},
	},
	Action: func(ctx *cli.Context) error {
		if errDaemon := daemon.EnsureRunning(ctx.Context); errDaemon != nil {
			return xerr.NewWM(errDaemon, "ensure daemon is running")
		}

		client, errList := client.NewGrpcClient()
		if errList != nil {
			return xerr.NewWM(errList, "new grpc client")
		}
		defer deferErr(client.Close)()

		app, errNewApp := pm.New(client)
		if errNewApp != nil {
			return xerr.NewWM(errNewApp, "new app")
		}

		command := ctx.Args().First()
		commandArgs := ctx.Args().Tail()
		name := ctx.String("name")
		tags := ctx.StringSlice("tag")
		workDir := ctx.String("pwd")
		if !ctx.IsSet("config") {
			if ctx.Args().Len() == 0 {
				return xerr.NewM("command expected")
			}

			if !ctx.IsSet("cwd") {
				cwd, err := os.Getwd()
				if err != nil {
					return xerr.NewWM(err, "get cwd")
				}
				workDir = cwd
			}

			runConfig := core.RunConfig{
				Command: command,
				Args:    commandArgs,
				Name:    fun.Optional(name, name != ""),
				Tags:    tags,
				Cwd:     workDir,
				Env:     nil,
			}

			procIDs, errRun := app.Run(ctx.Context, runConfig)
			fmt.Println(lo.ToAnySlice(procIDs)...)
			if errRun != nil {
				return xerr.NewWM(errRun, "run command", xerr.Fields{"runConfig": runConfig, "procIDs": procIDs})
			}

			return nil
		}

		configs, errLoadConfigs := core.LoadConfigs(ctx.String("config"))
		if errLoadConfigs != nil {
			return xerr.NewWM(errLoadConfigs, "load run configs")
		}

		names := ctx.Args().Slice()
		if len(names) == 0 {
			// no filtering by names, run all processes
			procIDs, err := app.Run(ctx.Context, configs...)
			fmt.Println(lo.ToAnySlice(procIDs)...)
			if err != nil {
				return xerr.NewWM(err, "create all procs from config", xerr.Fields{"created procIDs": procIDs})
			}

			return nil
		}

		configsByName := make(map[string]core.RunConfig, len(names))
		for _, cfg := range configs {
			if name, ok := cfg.Name.Unpack(); !ok || !lo.Contains(names, name) {
				continue
			}

			configsByName[cfg.Name.Unwrap()] = cfg
		}

		merr := xerr.Combine(lo.FilterMap(names, func(name string, _ int) (error, bool) {
			if _, ok := configsByName[name]; !ok {
				return xerr.NewM("unknown proc name", xerr.Fields{"name": name}), true
			}

			return nil, false
		})...)
		if merr != nil {
			return merr
		}

		procIDs, errCreate := app.Run(ctx.Context, lo.Values(configsByName)...)
		fmt.Println(lo.ToAnySlice(procIDs)...)
		if errCreate != nil {
			return xerr.NewWM(errCreate, "run procs filtered by name from config", xerr.Fields{
				"names":           names,
				"created procIDs": procIDs,
			})
		}

		return nil
	},
}
