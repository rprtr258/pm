package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/set"

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
func (app App) Run(config core.RunConfig) (core.PMID, error) {
	command, errLook := exec.LookPath(config.Command)
	if errLook != nil {
		// if command is relative and failed to look it up, add workdir first
		if filepath.IsLocal(config.Command) {
			config.Command = filepath.Join(config.Cwd, config.Command)
		}

		command, errLook = exec.LookPath(config.Command)
		if errLook != nil {
			return "", errors.Wrapf(errLook, "look for executable path: %q", config.Command)
		}
	}

	if command == config.Command { // command contains slash and might be relative
		var errAbs error
		command, errAbs = filepath.Abs(command)
		if errAbs != nil {
			return "", errors.Wrapf(errAbs, "get absolute binary path: %q", command)
		}
	}

	id, errCreate := func() (core.PMID, error) {
		watch := fun.OptMap(config.Watch, func(r *regexp.Regexp) string {
			return r.String()
		})
		// try to find by name and update
		if name, ok := config.Name.Unpack(); ok { //nolint:nestif // no idea how to simplify it now
			procs, err := app.DB.GetProcs(core.WithAllIfNoFilters)
			if err != nil {
				return "", errors.Wrapf(err, "get procs from db")
			}

			if procID, ok := fun.FindKeyBy(procs, func(_ core.PMID, procData core.Proc) bool {
				return procData.Name == name
			}); ok {
				procData := core.Proc{
					ID:         procID,
					Status:     core.NewStatusCreated(),
					Name:       name,
					Cwd:        config.Cwd,
					Tags:       fun.Uniq(append(config.Tags, "all")...),
					Command:    command,
					Args:       config.Args,
					Watch:      watch,
					Env:        config.Env,
					StdoutFile: config.StdoutFile.OrDefault(filepath.Join(app.DirLos, fmt.Sprintf("%v.stdout", procID))),
					StderrFile: config.StderrFile.OrDefault(filepath.Join(app.DirLos, fmt.Sprintf("%v.stderr", procID))),
					Startup:    config.Startup,
				}

				proc := procs[procID]
				if proc.Status.Status != core.StatusRunning ||
					proc.Cwd == procData.Cwd &&
						compareTags(proc.Tags, procData.Tags) &&
						proc.Command == procData.Command &&
						compareArgs(proc.Args, procData.Args) &&
						proc.Watch == procData.Watch {
					// not updated, do nothing
					return procID, nil
				}

				if errUpdate := app.DB.UpdateProc(procData); errUpdate != nil {
					return "", errors.Wrapf(errUpdate, "update proc: %v", procFields(procData))
				}

				return procID, nil
			}
		}

		procID, err := app.DB.AddProc(db.CreateQuery{
			Name:       config.Name.OrDefault(namegen.New()),
			Cwd:        config.Cwd,
			Tags:       fun.Uniq(append(config.Tags, "all")...),
			Command:    command,
			Args:       config.Args,
			Watch:      watch,
			Env:        config.Env,
			StdoutFile: config.StdoutFile,
			StderrFile: config.StderrFile,
		}, app.DirLos)
		if err != nil {
			return "", errors.Wrapf(err, "save proc")
		}

		return procID, nil
	}()
	if errCreate != nil {
		return "", errors.Wrapf(errCreate, "server.create: %v", config)
	}

	app.startAgent(id)

	return core.PMID(id), nil
}
