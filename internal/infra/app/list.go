package app

import (
	"github.com/rprtr258/fun/iter"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func (app App) List() iter.Seq[core.Proc] {
	procs, err := app.DB.GetProcs(core.WithAllIfNoFilters)
	if err != nil {
		return iter.FromNothing[core.Proc]()
	}

	for id, proc := range procs {
		if proc.Status.Status != core.StatusRunning {
			continue
		}

		stat, ok := linuxprocess.StatPMID(proc.ID, EnvPMID)
		if !ok {
			proc.Status = core.NewStatusStopped(-1)
			app.DB.StatusSet(id, proc.Status)
		}

		proc.Status.Memory = stat.Memory
		proc.Status.CPU = uint64(stat.CPU)

		procs[id] = proc
	}
	return iter.Values(iter.FromDict(procs))
}
