package cli

import (
	"fmt"
	"os"
	"regexp"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
)

type _cmdRun struct {
	Name *string  `short:"n" long:"name" description:"set a name for the process"`
	Tags []string `short:"t" long:"tag" description:"add specified tag"`
	Cwd  *string  `long:"cwd" description:"set working directory"`
	configFlag
	Watch *string `long:"watch" description:"restart on changes to files matching specified regex"`
	Args  struct {
		Command string
		Args    []string
	} `positional-args:"yes" required:"true"`
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

func (x *_cmdRun) Execute(_ []string) error {
	app, errNewApp := app.New()
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "new app")
	}

	if x.Config == nil {
		command := x.Args.Command
		commandArgs := x.Args.Args
		tags := x.Tags
		var workDir string
		if x.Cwd == nil {
			cwd, err := os.Getwd()
			if err != nil {
				return xerr.NewWM(err, "get cwd")
			}
			workDir = cwd
		} else {
			workDir = *x.Cwd
		}

		var watch fun.Option[*regexp.Regexp]
		if pattern := x.Watch; pattern != nil {
			watchRE, errCompile := regexp.Compile(*pattern)
			if errCompile != nil {
				return xerr.NewWM(errCompile, "compile watch regex", xerr.Fields{"pattern": *pattern})
			}

			watch = fun.Valid(watchRE)
		}

		runConfig := core.RunConfig{
			Command:    command,
			Args:       commandArgs,
			Name:       fun.FromPtr(x.Name),
			Tags:       tags,
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

		procID, errRun := app.Run(runConfig)
		if errRun != nil {
			return xerr.NewWM(errRun, "run command", xerr.Fields{
				"run_config": runConfig,
				"pmid":       procID,
			})
		}

		fmt.Println(procID)

		return nil
	}

	configs, errLoadConfigs := core.LoadConfigs(string(*x.configFlag.Config))
	if errLoadConfigs != nil {
		return xerr.NewWM(errLoadConfigs, "load run configs")
	}

	// TODO: if config is specified Args.Command and Args.Args are not required
	names := append(x.Args.Args, x.Args.Command)
	if len(names) == 0 {
		// no filtering by names, run all processes
		for _, config := range configs {
			procID, err := app.Run(config)
			fmt.Println(procID)
			if err != nil {
				fmt.Println()
				return xerr.NewWM(err, "create all procs from config", xerr.Fields{"pmid": procID})
			}
		}

		return nil
	}

	configsByName := make(map[string]core.RunConfig, len(names))
	for _, cfg := range configs {
		name, ok := cfg.Name.Unpack()
		if !ok || !fun.Contains(names, name) {
			continue
		}

		configsByName[name] = cfg
	}

	merr := xerr.Combine(fun.Map[error](names, func(name string) error {
		if _, ok := configsByName[name]; !ok {
			return xerr.NewM("unknown proc name", xerr.Fields{"name": name})
		}

		return nil
	})...)
	if merr != nil {
		return merr
	}

	for _, config := range configsByName {
		id, errCreate := app.Run(config)
		fmt.Println(id)
		if errCreate != nil {
			return xerr.NewWM(errCreate, "run procs filtered by name from config", xerr.Fields{
				"names": names,
				"pmid":  id,
			})
		}
	}

	return nil
}
