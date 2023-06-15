package daemon

import (
	"context"
	"time"

	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

type cron struct {
	l                 log.Logger
	db                db.Handle
	statusUpdateDelay time.Duration
}

func (c cron) updateStatuses() {
	for _, proc := range c.db.List() {
		if proc.Status.Status != db.StatusRunning {
			continue
		}

		_, errStat := linuxprocess.ReadProcessStat(proc.Status.Pid)
		switch {
		case errStat == nil:
			// process stat file exists hence process is still running
			continue
		case !xerr.Is(errStat, linuxprocess.ErrStatFileNotFound):
			c.l.Warnf("read proc stat", log.F{
				"pid": proc.Status.Pid,
				"err": errStat.Error(),
			})
		default:
			c.l.Infof("process seems to be stopped, updating status...", log.F{"pid": proc.Status.Pid})

			if errUpdate := c.db.SetStatus(proc.ProcID, db.NewStatusStopped(-1)); errUpdate != nil {
				c.l.Errorf("set stopped status", log.F{"procID": proc.ID})
			}
		}
	}
}

func (c cron) start(ctx context.Context) {
	ticker := time.NewTicker(c.statusUpdateDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.updateStatuses()
		case <-ctx.Done():
			c.l.Info("context canceled, stopping...")
			return
		}
	}
}