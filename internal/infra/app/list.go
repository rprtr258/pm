package app

import "github.com/rprtr258/pm/internal/core"

func (app App) List() map[core.PMID]core.Proc {
	procs, err := app.db.GetProcs(core.WithAllIfNoFilters)
	if err != nil {
		return nil
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
	return procs
}
