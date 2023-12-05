package app

import (
	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/pm/internal/core"
)

func (app App) List() iter.Seq[core.Proc] { // TODO: return iterator
	procs, err := app.db.GetProcs(core.WithAllIfNoFilters)
	if err != nil {
		return iter.FromNothing[core.Proc]()
	}

	for id, proc := range procs {
		if proc.Status.Status != core.StatusRunning {
			continue
		}

		// TODO: uncomment
		// if _, err := linuxprocess.ReadProcessStat(proc.PMID); err != nil {
		// 	proc.Status = core.NewStatusStopped()
		// 	if errSet := s.db.SetStatus(id, proc.Status); errSet != nil {
		// 		log.Error().Err(errSet).Msg("failed to update status to stopped")
		// 	}
		// }
		procs[id] = proc
	}
	return iter.Values(iter.FromDict(procs))
}

func (app App) Get(id core.PMID) (core.Proc, bool) {
	procs, err := app.db.GetProcs(core.WithIDs(id))
	if err != nil {
		return fun.Zero[core.Proc](), false
	}

	if proc, ok := procs[id]; ok {
		return proc, true
	}

	return fun.Zero[core.Proc](), false
}
