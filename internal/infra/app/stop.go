package app

import (
	"errors"
	"os"
	"syscall"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func (app App) stop(id core.PMID) error {
	{
		proc, ok := app.db.GetProc(id)
		if !ok {
			return xerr.NewM("not found proc to stop")
		}

		if proc.Status.Status != core.StatusRunning {
			return nil
		}
	}

	l := log.With().Stringer("pmid", id).Logger()

	proc, ok := linuxprocess.StatPMID(id, _envPMID)
	if !ok {
		return xerr.NewM("find process")
	}

	if errKill := syscall.Kill(-proc.Pid, syscall.SIGTERM); errKill != nil {
		switch {
		case errors.Is(errKill, os.ErrProcessDone):
			l.Warn().Msg("tried stop process which is done")
		case errors.Is(errKill, syscall.ESRCH): // no such process
			l.Warn().Msg("tried stop process which doesn't exist")
		default:
			return xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": proc.Pid})
		}
	}

	return nil
}

func (app App) Stop(ids ...core.PMID) error {
	for _, id := range ids {
		if errStop := app.stop(id); errStop != nil {
			log.Error().
				Err(errStop).
				Stringer("pmid", id).
				Msg("failed to stop proc")
		}
	}

	return nil
}
