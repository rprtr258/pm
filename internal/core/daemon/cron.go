package daemon

import (
	"context"
	"time"

	"golang.org/x/exp/slog"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

type cron struct {
	l                 *slog.Logger
	db                db.Handle
	statusUpdateDelay time.Duration
}

func (c cron) updateStatuses() {
	for procID, proc := range c.db.List() {
		if proc.Status.Status != core.StatusRunning {
			continue
		}

		switch _, errStat := linuxprocess.ReadProcessStat(proc.Status.Pid); errStat {
		case nil:
			// process stat file exists hence process is still running
			continue
		case linuxprocess.ErrStatFileNotFound:
			c.l.Info("process seems to be stopped, updating status...", "pid", proc.Status.Pid)

			if errUpdate := c.db.SetStatus(proc.ProcID, core.NewStatusStopped(-1)); errUpdate != nil {
				c.l.Error("set stopped status", "proc_id", procID)
			}
		default:
			c.l.Warn("read proc stat",
				"pid", proc.Status.Pid,
				"err", errStat.Error(),
			)
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
