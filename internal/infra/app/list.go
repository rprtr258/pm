package app

import (
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/process"

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
			if errSet := app.DB.SetStatus(id, proc.Status); errSet != nil {
				log.Error().Err(errSet).Msg("failed to update status to stopped")
			}
		}

		if p, err := process.NewProcess(int32(stat.Pid)); err == nil {
			totalMemory := uint64(0)
			totalCPU := float64(0)
			children, _ := p.Children()
			for _, child := range children {
				if mem, err := child.MemoryInfo(); err == nil {
					totalMemory += mem.RSS
				}
				if cpu, err := child.CPUPercent(); err == nil {
					totalCPU = cpu
				}
			}
			proc.Status.Memory = totalMemory
			proc.Status.CPU = uint64(totalCPU)
		}

		procs[id] = proc
	}
	return iter.Values(iter.FromDict(procs))
}
