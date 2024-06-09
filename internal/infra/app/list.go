package app

import (
	"time"

	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func (app App) List() iter.Seq[core.ProcStat] {
	procs, err := app.DB.GetProcs(core.WithAllIfNoFilters)
	if err != nil {
		log.Error().Err(err).Msg("get procs")
		return iter.FromNothing[core.ProcStat]()
	}

	list := linuxprocess.List()
	return func(yield func(core.ProcStat) bool) {
		for _, proc := range procs {
			var procStat core.ProcStat
			stat, ok := linuxprocess.StatPMID(list, proc.ID, EnvPMID)
			switch {
			case !ok: // no shim at all
				procStat = core.ProcStat{
					Proc:      proc,
					Status:    core.StatusStopped,
					StartTime: time.Time{}, CPU: 0, Memory: 0,
				}
			case stat.ChildStartTime.IsZero(): // shim is running but no child
				procStat = core.ProcStat{
					Proc:      proc,
					Status:    core.StatusCreated,
					StartTime: time.Time{}, CPU: 0, Memory: 0,
				}
			default: // shim is running and child is happy too
				procStat = core.ProcStat{
					Proc:      proc,
					StartTime: stat.ChildStartTime,
					CPU:       stat.CPU,
					Memory:    stat.Memory,
					Status:    core.StatusRunning,
				}
			}
			yield(procStat)
		}
	}
}
