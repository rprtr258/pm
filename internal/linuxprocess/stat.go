package linuxprocess

import (
	"math"
	"time"

	"github.com/rprtr258/fun"

	"github.com/rprtr258/pm/internal/core"
)

type Stat struct {
	ShimPID int
	Memory  uint64  // bytes
	CPU     float64 // percent

	// might be zero
	ChildPID       int
	ChildStartTime time.Time
}

// Returns whole children subtree as list.
// list must be sorted by pid
func Children(list []ProcListItem, pid int) []ProcListItem {
	pids := map[int]struct{}{pid: {}}
	res := []ProcListItem{}
	for _, p := range list {
		ppid, _ := p.P.Ppid()
		if _, ok := pids[int(ppid)]; ok {
			res = append(res, p)
			pids[p.Handle.Pid] = struct{}{}
		}
	}
	return res
}

// list must be sorted by pid
func StatPMID(list []ProcListItem, pmid core.PMID) (Stat, bool) {
	shim, _, ok := fun.Index(func(p ProcListItem) bool {
		return p.Environ[core.EnvPMID] == string(pmid)
	}, list...)
	if !ok {
		return fun.Zero[Stat](), false
	}

	children := Children(list, shim.Handle.Pid)
	if len(children) == 0 {
		// no children, no stats
		return Stat{
			ShimPID:        shim.Handle.Pid,
			Memory:         0,
			CPU:            0,
			ChildPID:       0,
			ChildStartTime: time.Time{},
		}, true
	}

	totalMemory := uint64(0)
	totalCPU := float64(0)
	startTimeUnix := int64(math.MaxInt64)
	for _, child := range children {
		if mem, err := child.P.MemoryInfo(); err == nil {
			totalMemory += mem.RSS
		}
		if cpu, err := child.P.CPUPercent(); err == nil {
			totalCPU = cpu
		}

		// find oldest child process
		if startUnix, err := child.P.CreateTime(); err == nil && startUnix < startTimeUnix {
			startTimeUnix = startUnix
		}
	}
	return Stat{
		ShimPID:        shim.Handle.Pid,
		Memory:         totalMemory,
		CPU:            totalCPU,
		ChildPID:       children[0].Handle.Pid,
		ChildStartTime: time.Unix(0, startTimeUnix*time.Millisecond.Nanoseconds()),
	}, true
}
