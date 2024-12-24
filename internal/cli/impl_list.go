package cli

import (
	"iter"
	"maps"
	"slices"

	"github.com/rprtr258/fun"
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
	return procSeq{func(yield func(core.ProcStat) bool) {
		for proc := range l.Seq {
			if p(proc) && !yield(proc) {
				break
			}
		}
	}}
}

func (l procSeq) Slice() []core.ProcStat {
	return slices.Collect(l.Seq)
}

func (l procSeq) IDs() iter.Seq[core.PMID] {
	return func(yield func(core.PMID) bool) {
		for ps := range l.Seq {
			if !yield(ps.ID) {
				break
			}
		}
	}
}

func (l procSeq) Tags() iter.Seq[string] {
	tags := map[string]struct{}{"all": {}}
	for ps := range l.Seq {
		for _, tag := range ps.Tags {
			tags[tag] = struct{}{}
		}
	}
	return maps.Keys(tags)
}

func listProcs(db db.Handle) procSeq {
	procs, err := db.GetProcs(core.WithAllIfNoFilters)
	if err != nil {
		log.Error().Err(err).Msg("get procs")
		return procSeq{nil}
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
			if !yield(procStat) {
				return
			}
		}
	}}
}
