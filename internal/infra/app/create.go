package app

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/fun/set"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/namegen"
	"github.com/rprtr258/pm/internal/infra/db"
)

// compareTags and return true if equal
func compareTags(first, second []string) bool {
	firstSet := set.NewFrom(first...)
	secondSet := set.NewFrom(second...)
	return set.Intersect(firstSet, secondSet).Size() == max(firstSet.Size(), secondSet.Size())
}

// compareArgs and return true if equal
func compareArgs(first, second []string) bool {
	return len(first) == len(second) && iter.FromSlice(first).All(func(iv fun.Pair[int, string]) bool {
		i, v := iv.K, iv.V
		return v == second[i]
	})
}

func (app App) create(
	command string,
	args []string,
	name fun.Option[string],
	cwd string,
	tags []string,
	env map[string]string,
	watch fun.Option[string],
	stdoutFile fun.Option[string],
	stderrFile fun.Option[string],
) (core.PMID, error) {
	// try to find by name and update
	if name, ok := name.Unpack(); ok { //nolint:nestif // no idea how to simplify it now
		procs := app.db.GetProcs(core.WithAllIfNoFilters)

		if procID, ok := fun.FindKeyBy(procs, func(_ core.PMID, procData core.Proc) bool {
			return procData.Name == name
		}); ok {
			procData := core.Proc{
				ID:         procID,
				Status:     core.NewStatusCreated(),
				Name:       name,
				Cwd:        cwd,
				Tags:       fun.Uniq(append(tags, "all")),
				Command:    command,
				Args:       args,
				Watch:      watch,
				Env:        env,
				StdoutFile: stdoutFile.OrDefault(filepath.Join(app.logsDir, fmt.Sprintf("%v.stdout", procID))),
				StderrFile: stderrFile.OrDefault(filepath.Join(app.logsDir, fmt.Sprintf("%v.stderr", procID))),
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

			if errUpdate := app.db.UpdateProc(procData); errUpdate != nil {
				return "", xerr.NewWM(errUpdate, "update proc", xerr.Fields{
					// "procData": procFields(procData),
				})
			}

			return procID, nil
		}
	}

	procID, err := app.db.AddProc(db.CreateQuery{
		Name:       name.OrDefault(namegen.New()),
		Cwd:        cwd,
		Tags:       fun.Uniq(append(tags, "all")),
		Command:    command,
		Args:       args,
		Watch:      watch,
		Env:        env,
		StdoutFile: stdoutFile,
		StderrFile: stderrFile,
	}, app.logsDir)
	if err != nil {
		return "", xerr.NewWM(err, "save proc")
	}

	return procID, nil
}

func (app App) Create(req core.RunConfig) (core.PMID, error) {
	procID, err := app.create(
		req.Command,
		req.Args,
		req.Name,
		req.Cwd,
		req.Tags,
		req.Env,
		fun.OptMap(req.Watch, func(r *regexp.Regexp) string {
			return r.String()
		}),
		req.StdoutFile,
		req.StderrFile,
	)
	if err != nil {
		return "", xerr.NewWM(err, "server.create")
	}

	return procID, nil
}
