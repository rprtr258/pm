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
	for procID, proc := range c.db.GetProcs(core.WithAllIfNoFilters) {
		if proc.Status.Status != core.StatusRunning {
			continue
		}

		switch _, errStat := linuxprocess.ReadProcessStat(proc.Status.Pid); errStat {
		case nil:
			// process stat file exists hence process is still running
			continue
		case linuxprocess.ErrStatFileNotFound:
			c.l.Info("process seems to be stopped, updating status...", "pid", proc.Status.Pid)

			// TODO: unsubscribe from watch
			if errUpdate := c.db.SetStatus(procID, core.NewStatusStopped(-1)); errUpdate != nil {
				c.l.Error("set stopped status", "proc_id", procID)
			}
		default:
			c.l.Warn(
				"read proc stat",
				slog.Int("pid", proc.Status.Pid),
				slog.Any("err", errStat),
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
