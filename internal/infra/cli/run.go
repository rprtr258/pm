package cli

import (
	"fmt"
	"os"
	"regexp"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
)

func run(app app.App, configs iter.Seq[core.RunConfig]) error {
	var merr error
	configs(func(config core.RunConfig) bool {
		id, errRun := app.Run(config)
		if errRun != nil {
			log.Error().
				Err(errRun).
				Dict("config", zerolog.Dict().
					Any("name", config.Name).
					Str("command", config.Command).
					Strs("args", config.Args).
					Str("cwd", config.Cwd).
					Strs("tags", config.Tags).
					Any("watch", config.Watch).
					Any("env", config.Env).
					Any("stdout_file", config.StdoutFile).
					Any("stderr_file", config.StderrFile),
				).
				Msg("failed to run proc")
			merr = errors.Newf("failed to start some procs")
		} else {
			fmt.Println(id)
		}

		return true
	})
	return merr
}

var _cmdRun = func() *cobra.Command {
	var name, cwd, config, watch string
	var tags []string
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "create and run new process",
		GroupID: "management",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := fun.IF(cmd.Flags().Lookup("name").Changed, &name, nil)
			cwd := fun.IF(cmd.Flags().Lookup("cwd").Changed, &cwd, nil)
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)
			watch := fun.IF(cmd.Flags().Lookup("watch").Changed, &watch, nil)

			app, errNewApp := app.New()
			if errNewApp != nil {
				return errors.Wrapf(errNewApp, "new app")
			}

			if config == nil {
				if len(args) == 0 {
					return errors.Newf("neither command nor config specified")
				}
				command, args := args[0], args[1:]

				var workDir string
				if cwd == nil {
					cwd, err := os.Getwd()
					if err != nil {
						return errors.Wrapf(err, "get cwd")
					}
					workDir = cwd
				} else {
					workDir = *cwd
				}

				var watchOpt fun.Option[*regexp.Regexp]
				if pattern := watch; pattern != nil {
					watchRE, errCompile := regexp.Compile(*pattern)
					if errCompile != nil {
						return errors.Wrapf(errCompile, "compile watch regex: %q", *pattern)
					}

					watchOpt = fun.Valid(watchRE)
				}

				runConfig := core.RunConfig{
					Command:    command,
					Args:       args,
					Name:       fun.FromPtr(name),
					Tags:       tags,
					Cwd:        workDir,
					Env:        nil,
					Watch:      watchOpt,
					StdoutFile: fun.Invalid[string](),
					StderrFile: fun.Invalid[string](),
					Actions: core.Actions{
						Healthcheck: nil,
						Custom:      nil,
					},
					KillTimeout:  0,
					KillChildren: false,
					Autorestart:  false,
					MaxRestarts:  0,
				}

				return run(app, iter.FromMany(runConfig))
			}

			configs, errLoadConfigs := core.LoadConfigs(*config)
			if errLoadConfigs != nil {
				return errors.Wrapf(errLoadConfigs, "load run configs")
			}

			// TODO: if config is specified Args.Command and Args.Args are not required
			names := args
			if len(names) == 0 {
				// no filtering by names, run all processes
				return run(app, iter.FromMany(configs...))
			}

			configsByName := make(map[string]core.RunConfig, len(names))
			for _, cfg := range configs {
				name, ok := cfg.Name.Unpack()
				if !ok || !fun.Contains(name, names...) {
					continue
				}

				configsByName[name] = cfg
			}

			merr := errors.Combine(fun.Map[error](func(name string) error {
				if _, ok := configsByName[name]; !ok {
					return errors.Newf("unknown proc name: %q", name)
				}

				return nil
			}, names...)...)
			if merr != nil {
				return merr
			}

			return run(app, iter.Values(iter.FromDict(configsByName)))
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "set a name for the process")
	cmd.Flags().StringSliceVarP(&tags, "tag", "t", nil, "add specified tag")
	cmd.Flags().StringVar(&cwd, "cwd", "", "set working directory")
	addFlagConfig(cmd, &config)
	cmd.Flags().StringVar(&watch, "watch", "", "restart on changes to files matching specified regex")
	// TODO: not yet implemented run parameters
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
	return cmd
}()
