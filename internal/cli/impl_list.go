package cli

import (
	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/pm/internal/linuxprocess"
)

// procSeq iterator with custom methods
type procSeq struct {
	iter.Seq[core.ProcStat]
}

func (l procSeq) Filter(p func(core.ProcStat) bool) procSeq {
	return procSeq{l.Seq.Filter(p)}
}

func (l procSeq) IDs() iter.Seq[core.PMID] {
	return iter.Map(l.Seq, func(proc core.ProcStat) core.PMID {
		return proc.ID
	})
}

func (l procSeq) Tags() iter.Seq[string] {
	tags := map[string]struct{}{"all": {}}
	l.ForEach(func(ps core.ProcStat) {
		for _, tag := range ps.Tags {
			tags[tag] = struct{}{}
		}
	})
	return iter.Keys(iter.FromDict(tags))
}

func listProcs(db db.Handle) procSeq {
	procs, err := db.GetProcs(core.WithAllIfNoFilters)
	if err != nil {
		log.Error().Err(err).Msg("get procs")
		return procSeq{iter.FromNothing[core.ProcStat]()}
	}

	list := linuxprocess.List()
	return procSeq{func(yield func(core.ProcStat) bool) {
		for _, proc := range procs {
			stat, ok := linuxprocess.StatPMID(list, proc.ID)
			procStat := core.ProcStat{ //nolint:exhaustruct // filled in switch below
				Proc:    proc,
				ShimPID: stat.ShimPID,
			}
			switch {
			case !ok: // no shim at all
				procStat.Status = core.StatusStopped
			case stat.ChildStartTime.IsZero(): // shim is running but no child
				procStat.Status = core.StatusCreated
			default: // shim is running and child is happy too
				procStat.StartTime = stat.ChildStartTime
				procStat.CPU = stat.CPU
				procStat.Memory = stat.Memory
				procStat.Status = core.StatusRunning
				procStat.ChildPID = fun.Valid(stat.ChildPID)
			}
			yield(procStat)
		}
	}}
}
