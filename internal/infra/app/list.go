package app

import (
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"

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

		if _, ok := linuxprocess.StatPMID(proc.ID, EnvPMID); !ok {
			proc.Status = core.NewStatusStopped(-1)
			if errSet := app.DB.SetStatus(id, proc.Status); errSet != nil {
				log.Error().Err(errSet).Msg("failed to update status to stopped")
			}
		}
		procs[id] = proc
	}
	return iter.Values(iter.FromDict(procs))
}
