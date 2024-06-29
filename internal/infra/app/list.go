package app

import (
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

type ProcList struct {
	iter.Seq[core.ProcStat]
}

func (l ProcList) Filter(p func(core.ProcStat) bool) ProcList {
	return ProcList{l.Seq.Filter(p)}
}

func (l ProcList) IDs() iter.Seq[core.PMID] {
	return iter.Map(l.Seq, func(proc core.ProcStat) core.PMID {
		return proc.ID
	})
}

func (l ProcList) Tags() iter.Seq[string] {
	tags := map[string]struct{}{"all": {}}
	l.ForEach(func(ps core.ProcStat) {
		for _, tag := range ps.Tags {
			tags[tag] = struct{}{}
		}
	})
	return iter.Keys(iter.FromDict(tags))
}

func (app App) List() ProcList {
	procs, err := app.DB.GetProcs(core.WithAllIfNoFilters)
	if err != nil {
		log.Error().Err(err).Msg("get procs")
		return ProcList{iter.FromNothing[core.ProcStat]()}
	}

	list := linuxprocess.List()
	return ProcList{func(yield func(core.ProcStat) bool) {
		for _, proc := range procs {
			var procStat core.ProcStat
			stat, ok := linuxprocess.StatPMID(list, proc.ID, EnvPMID)
			switch {
			case !ok: // no shim at all
				procStat = core.ProcStat{
					Proc:      proc,
					ShimPID:   stat.ShimPID,
					Status:    core.StatusStopped,
					StartTime: time.Time{}, CPU: 0, Memory: 0, ChildPID: fun.Invalid[int](),
				}
			case stat.ChildStartTime.IsZero(): // shim is running but no child
				procStat = core.ProcStat{
					Proc:      proc,
					ShimPID:   stat.ShimPID,
					Status:    core.StatusCreated,
					StartTime: time.Time{}, CPU: 0, Memory: 0, ChildPID: fun.Invalid[int](),
				}
			default: // shim is running and child is happy too
				procStat = core.ProcStat{
					Proc:      proc,
					ShimPID:   stat.ShimPID,
					StartTime: stat.ChildStartTime,
					CPU:       stat.CPU,
					Memory:    stat.Memory,
					Status:    core.StatusRunning,
					ChildPID:  fun.Valid(stat.ChildPID),
				}
			}
			yield(procStat)
		}
	}}
}
