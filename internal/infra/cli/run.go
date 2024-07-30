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
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/fun/set"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/namegen"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
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

// runProc - create and start processes, returns ids of created processes.
// ids must be handled before handling error, because it tries to run all
// processes and error contains info about all failed processes, not only first.
func runProc(
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

	name := config.Name

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

			// proc updated, if it is running, stop it to start later
			if _, ok := linuxprocess.StatPMID(dbb.ListRunning(), procID); ok {
				if err := implStop(dbb, procID); err != nil {
					return "", errors.Wrapf(err, "stop updated proc: %v", procID)
				}
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

func runProcs(db db.Handle, dirLogs string, configs ...core.RunConfig) error {
	// depends_on validation
	{
		// collect all names from db and configs list
		allNames := set.NewFrom(iter.Map(listProcs(dbb).Seq, func(ps core.ProcStat) string {
			return ps.Name
		}).ToSlice()...)
		for _, config := range configs {
			allNames.Add(config.Name)
		}

		nonexistingErrors := []error{}
		for _, config := range configs {
			nonexistingDepends := []string{}
			for _, name := range config.DependsOn {
				if !allNames.Contains(name) {
					nonexistingDepends = append(nonexistingDepends, name)
				}
			}
			if len(nonexistingDepends) > 0 {
				nonexistingErrors = append(nonexistingErrors, errors.Newf(
					"%q depends on non-existing processes: %v",
					config.Name, nonexistingDepends))
			}
		}
		if len(nonexistingErrors) > 0 {
			return errors.Combine(nonexistingErrors...)
		}
	}

	// sort procs by depends_on
	{
		indexByName := fun.SliceToMap[string, int](
			func(proc core.RunConfig, i int) (string, int) { return proc.Name, i },
			configs...)

		// topological sort
		type visitStatus int8
		const (
			statusNotVisited visitStatus = iota
			statusInProgress
			statusProcessed
		)
		loopFound := false
		res := []int{} // indices in configs slice
		visited := make([]visitStatus, len(configs))
		var dfs func(int)
		dfs = func(i int) {
			switch visited[i] {
			case statusInProgress:
				loopFound = true
				log.Error().
					Strs("loop", fun.Map[string](func(i int) string { return configs[i].Name }, res...)).
					Msg("loop found")
			case statusProcessed:
			case statusNotVisited:
				visited[i] = statusInProgress
				for _, dependency := range configs[i].DependsOn {
					dfs(indexByName[dependency])
					if loopFound {
						return
					}
				}
				res = append(res, i)
				visited[i] = statusProcessed
			}
		}
		for i := 0; i < len(configs); i++ {
			dfs(i)
			if loopFound {
				return errors.Newf("loop found")
			}
		}

		// actually sort procs by indices in res slice
		configs = fun.Map[core.RunConfig](func(i int) core.RunConfig { return configs[i] }, res...)
	}

	var merr []error
	for _, config := range configs {
		if _, name, errRun := runProc(db, dirLogs, config); errRun != nil {
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
	var maxRestarts uint
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "create and run new process",
		GroupID: "management",
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			name := fun.IF(cmd.Flags().Lookup("name").Changed, &name, nil)
			cwd := fun.IF(cmd.Flags().Lookup("cwd").Changed, &cwd, nil)
			config := fun.IF(cmd.Flags().Lookup("config").Changed, &config, nil)
			watch := fun.IF(cmd.Flags().Lookup("watch").Changed, &watch, nil)

			if config == nil { // inline run, e.g. `pm run -- npm dev`
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
					Name:        fun.FromPtr(name).OrDefault(namegen.New()),
					Tags:        tags,
					Cwd:         workDir,
					Env:         nil,
					Watch:       watchOpt,
					StdoutFile:  fun.Invalid[string](),
					StderrFile:  fun.Invalid[string](),
					KillTimeout: 0,
					Autorestart: false,
					MaxRestarts: maxRestarts,
					Startup:     false,
					DependsOn:   nil,
				}

				return runProcs(dbb, cfg.DirLogs, runConfig)
			}

			configs, errLoadConfigs := core.LoadConfigs(*config)
			if errLoadConfigs != nil {
				return errors.Wrapf(errLoadConfigs, "load run configs")
			}

			names := posArgs
			if len(names) == 0 {
				// no filtering by names, run all processes
				return runProcs(dbb, cfg.DirLogs, configs...)
			}

			configsByName := make(map[string]core.RunConfig, len(names))
			for _, cfg := range configs {
				name := cfg.Name
				if !fun.Contains(name, names...) {
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

			return runProcs(dbb, cfg.DirLogs, fun.Values(configsByName)...)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "set a name for the process")
	cmd.Flags().StringSliceVarP(&tags, "tag", "t", nil, "add specified tag")
	cmd.Flags().StringVar(&cwd, "cwd", "", "set working directory")
	addFlagConfig(cmd, &config)
	cmd.Flags().StringVar(&watch, "watch", "", "restart on changes to files matching specified regex")
	cmd.Flags().UintVar(&maxRestarts, "max-restarts", 0, "autorestart process, giving up after COUNT times")
	return cmd
}()
