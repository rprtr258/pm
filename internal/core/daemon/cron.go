package daemon

import (
	"context"
	"time"

	"github.com/rprtr258/log"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
	"github.com/rprtr258/xerr"
)

type cron struct {
	db                db.Handle
	statusUpdateDelay time.Duration
}

func (c cron) updateStatuses() {
	for _, proc := range c.db.List() {
		if proc.Status.Status != core.StatusRunning {
			continue
		}

		_, errStat := linuxprocess.ReadProcessStat(proc.Status.Pid)
		switch {
		case errStat == nil:
			// process stat file exists hence process is still running
			continue
		case !xerr.Is(errStat, linuxprocess.ErrStatFileNotFound):
			log.Warnf("read proc stat", log.F{
				"pid": proc.Status.Pid,
				"err": errStat.Error(),
			})
		default:
			log.Infof("process seems to be stopped, updating status...", log.F{"pid": proc.Status.Pid})

			if errUpdate := c.db.SetStatus(proc.ProcID, core.NewStatusStopped(-1)); errUpdate != nil {
				log.Errorf("set stopped status", log.F{"procID": proc.ID})
			}
		}
	}
}

func (c cron) start(ctx context.Context) {
	ticker := time.NewTicker(c.statusUpdateDelay)
	for {
		select {
		case <-ticker.C:
			c.updateStatuses()
		case <-ctx.Done():
			return
		}
	}
}
