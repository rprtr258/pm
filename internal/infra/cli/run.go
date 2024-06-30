package cli

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/set"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/namegen"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/errors"
)

// compareTags and return true if equal
func compareTags(first, second []string) bool {
	firstSet := set.NewFrom(first...)
	secondSet := set.NewFrom(second...)
	if firstSet.Size() != secondSet.Size() {
		return false
	}

	equal := true
	firstSet.Iter()(func(s string) bool {
		if !secondSet.Contains(s) {
			equal = false
			return false
		}
		return true
	})
	return equal
}

// compareArgs and return true if equal
func compareArgs(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}

	for i, v := range first {
		if v != second[i] {
			return false
		}
	}

	return true
}

// Run - create and start processes, returns ids of created processes.
// ids must be handled before handling error, because it tries to run all
// processes and error contains info about all failed processes, not only first.
func ImplRun(
	dbb db.Handle,
	dirLogs string,
	config core.RunConfig,
) (core.PMID, string, error) {
	command, errLook := exec.LookPath(config.Command)
	if errLook != nil {
		// if command is relative and failed to look it up, add workdir first
		if filepath.IsLocal(config.Command) {
			config.Command = filepath.Join(config.Cwd, config.Command)
		}

		command, errLook = exec.LookPath(config.Command)
		if errLook != nil {
			return "", "", errors.Wrapf(errLook, "look for executable path: %q", config.Command)
		}
	}

	if command == config.Command { // command contains slash and might be relative
		var errAbs error
		command, errAbs = filepath.Abs(command)
		if errAbs != nil {
			return "", "", errors.Wrapf(errAbs, "get absolute binary path: %q", command)
		}
	}

	name := config.Name.OrDefault(namegen.New())

	id, errCreate := func() (core.PMID, error) {
		watch := fun.OptMap(config.Watch, func(r *regexp.Regexp) string {
			return r.String()
		})
		// try to find by name and update
		procs, err := dbb.GetProcs(core.WithAllIfNoFilters)
		if err != nil {
			return "", errors.Wrapf(err, "get procs from db")
		}

		if procID, ok := fun.FindKeyBy(procs, func(_ core.PMID, procData core.Proc) bool {
			return procData.Name == name
		}); ok {
			procData := core.Proc{
				ID:          procID,
				Name:        name,
				Cwd:         config.Cwd,
				Tags:        fun.Uniq(append(config.Tags, "all")...),
				Command:     command,
				Args:        config.Args,
				Watch:       watch,
				Env:         config.Env,
				StdoutFile:  config.StdoutFile.OrDefault(filepath.Join(dirLogs, fmt.Sprintf("%v.stdout", procID))),
				StderrFile:  config.StderrFile.OrDefault(filepath.Join(dirLogs, fmt.Sprintf("%v.stderr", procID))),
				Startup:     config.Startup,
				KillTimeout: config.KillTimeout,
				DependsOn:   config.DependsOn,
			}

			proc := procs[procID]
			if proc.Cwd == procData.Cwd &&
				compareTags(proc.Tags, procData.Tags) &&
				proc.Command == procData.Command &&
				compareArgs(proc.Args, procData.Args) &&
				proc.Watch == procData.Watch {
				// not updated, do nothing
				return procID, nil
			}

			if errUpdate := dbb.UpdateProc(procData); errUpdate != nil {
				return "", errors.Wrapf(errUpdate, "update proc: %v", procData)
			}

			return procID, nil
		}

		procID, err := dbb.AddProc(db.CreateQuery{
			Name:        name,
			Cwd:         config.Cwd,
			Tags:        fun.Uniq(append(config.Tags, "all")...),
			Command:     command,
			Args:        config.Args,
			Watch:       watch,
			Env:         config.Env,
			StdoutFile:  config.StdoutFile,
			StderrFile:  config.StderrFile,
			Startup:     config.Startup,
			KillTimeout: cmp.Or(config.KillTimeout, 5*time.Second),
			DependsOn:   config.DependsOn,
		}, dirLogs)
		if err != nil {
			return "", errors.Wrapf(err, "save proc")
		}

		return procID, nil
	}()
	if errCreate != nil {
		return "", "", errors.Wrapf(errCreate, "server.create: %v", config)
	}

	err := implStart(dbb, id)
	return id, name, err
}

func run(db db.Handle, dirLogs string, configs ...core.RunConfig) error {
	var merr []error
	for _, config := range configs {
		if _, name, errRun := ImplRun(db, dirLogs, config); errRun != nil {
			merr = append(merr, errors.Wrapf(errRun, "start proc %v", config))
		} else {
			fmt.Println(name)
		}
	}
	return errors.Combine(merr...)
}

var _cmdRun = func() *cobra.Command {
	var name, cwd, config, watch string
	var tags []string
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "create and run new process",
		GroupID: "management",
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			name := fun.IF(cmd.Flags().Lookup("name").Changed, &name, nil)
			cwd := fun.IF(cmd.Flags().Lookup("cwd").Changed, &cwd, nil)
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)
			watch := fun.IF(cmd.Flags().Lookup("watch").Changed, &watch, nil)

			if config == nil {
				if len(posArgs) == 0 {
					return errors.Newf("neither command nor config specified")
				}
				command, args := posArgs[0], posArgs[1:]

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
					Command:     command,
					Args:        args,
					Name:        fun.FromPtr(name),
					Tags:        tags,
					Cwd:         workDir,
					Env:         nil,
					Watch:       watchOpt,
					StdoutFile:  fun.Invalid[string](),
					StderrFile:  fun.Invalid[string](),
					KillTimeout: 0,
					Autorestart: false,
					MaxRestarts: 0,
					Startup:     false,
					DependsOn:   nil,
				}

				return run(dbb, cfg.DirLogs, runConfig)
			}

			configs, errLoadConfigs := core.LoadConfigs(*config)
			if errLoadConfigs != nil {
				return errors.Wrapf(errLoadConfigs, "load run configs")
			}

			// TODO: if config is specified Args.Command and Args.Args are not required
			names := posArgs
			if len(names) == 0 {
				// no filtering by names, run all processes
				return run(dbb, cfg.DirLogs, configs...)
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

			return run(dbb, cfg.DirLogs, fun.Values(configsByName)...)
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
