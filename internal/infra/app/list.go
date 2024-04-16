package app

import (
	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func (app App) List() iter.Seq[core.Proc] {
	procs, err := app.db.GetProcs(core.WithAllIfNoFilters)
	if err != nil {
		return iter.FromNothing[core.Proc]()
	}

	for id, proc := range procs {
		if proc.Status.Status != core.StatusRunning {
			continue
		}

		if _, ok := linuxprocess.StatPMID(proc.ID, EnvPMID); !ok {
			proc.Status = core.NewStatusStopped(-1)
			if errSet := app.db.SetStatus(id, proc.Status); errSet != nil {
				log.Error().Err(errSet).Msg("failed to update status to stopped")
			}
		}
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
