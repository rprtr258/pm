package daemon

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

type cron struct {
	l                 zerolog.Logger
	db                db.Handle
	statusUpdateDelay time.Duration
	ebus              *eventbus.EventBus
}

func StartCron(ctx context.Context, ebus *eventbus.EventBus, dbHandle db.Handle) {
	cron{
		l:                 log.Logger.With().Str("system", "cron").Logger(),
		db:                dbHandle,
		statusUpdateDelay: 5 * time.Second,
		ebus:              ebus,
	}.start(ctx)
}

func (c cron) updateStatuses(ctx context.Context) {
	for procID, proc := range c.db.GetProcs(core.WithAllIfNoFilters) {
		if proc.Status.Status != core.StatusRunning {
			continue
		}

		switch _, errStat := linuxprocess.ReadProcessStat(proc.Status.Pid); errStat {
		case nil:
			// process stat file exists hence process is still running
			continue
		case linuxprocess.ErrStatFileNotFound:
			c.l.Info().
				Int("pid", proc.Status.Pid).
				Msg("process seems to be stopped, updating status...")

			c.ebus.Publish(ctx, eventbus.NewPublishProcStopped(procID, -1, eventbus.EmitReasonDied))
		default:
			c.l.Warn().
				Err(errStat).
				Int("pid", proc.Status.Pid).
				Msg("read proc stat")
		}
	}
}

func (c cron) start(ctx context.Context) {
	ticker := time.NewTicker(c.statusUpdateDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.updateStatuses(ctx)
		case <-ctx.Done():
			c.l.Info().Msg("context canceled, stopping...")
			return
		}
	}
}
