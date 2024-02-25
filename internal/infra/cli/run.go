package cli

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
			merr = errors.New("failed to start some procs")
		} else {
			fmt.Println(id)
		}

		return true
	})
	return merr
}

type _cmdRun struct {
	Name *string  `short:"n" long:"name" description:"set a name for the process"`
	Tags []string `short:"t" long:"tag" description:"add specified tag"`
	Cwd  *string  `long:"cwd" description:"set working directory"`
	configFlag
	Watch *string  `long:"watch" description:"restart on changes to files matching specified regex"`
	Args  []string `positional-args:"yes"`
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
}

func (x _cmdRun) Execute(ctx context.Context) error {
	app, errNewApp := app.New()
	if errNewApp != nil {
		return errors.Wrap(errNewApp, "new app")
	}

	if x.Config == nil {
		if len(x.Args) == 0 {
			return errors.New("neither command nor config specified")
		}
		command, args := x.Args[0], x.Args[1:]

		var workDir string
		if x.Cwd == nil {
			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "get cwd")
			}
			workDir = cwd
		} else {
			workDir = *x.Cwd
		}

		var watch fun.Option[*regexp.Regexp]
		if pattern := x.Watch; pattern != nil {
			watchRE, errCompile := regexp.Compile(*pattern)
			if errCompile != nil {
				return errors.Wrap(errCompile, "compile watch regex: %q", *pattern)
			}

			watch = fun.Valid(watchRE)
		}

		runConfig := core.RunConfig{
			Command:    command,
			Args:       args,
			Name:       fun.FromPtr(x.Name),
			Tags:       x.Tags,
			Cwd:        workDir,
			Env:        nil,
			Watch:      watch,
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

	configs, errLoadConfigs := core.LoadConfigs(string(*x.configFlag.Config))
	if errLoadConfigs != nil {
		return errors.Wrap(errLoadConfigs, "load run configs")
	}

	// TODO: if config is specified Args.Command and Args.Args are not required
	names := x.Args
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

	merr := xerr.Combine(fun.Map[error](func(name string) error {
		if _, ok := configsByName[name]; !ok {
			return errors.New("unknown proc name: %q", name)
		}

		return nil
	}, names...)...)
	if merr != nil {
		return merr
	}

	return run(app, iter.Values(iter.FromDict(configsByName)))
}
