package app

import (
	stdErrors "errors"
	"os"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func (app App) stop(id core.PMID) error {
	{
		proc, ok := app.db.GetProc(id)
		if !ok {
			return errors.Newf("not found proc to stop")
		}

		if proc.Status.Status != core.StatusRunning {
			return nil
		}
	}

	l := log.With().Stringer("pmid", id).Logger()

	proc, ok := linuxprocess.StatPMID(id, EnvPMID)
	if !ok {
		return errors.Newf("find process")
	}

	if errKill := syscall.Kill(-proc.Pid, syscall.SIGTERM); errKill != nil {
		switch {
		case stdErrors.Is(errKill, os.ErrProcessDone):
			l.Warn().Msg("tried stop process which is done")
		case stdErrors.Is(errKill, syscall.ESRCH): // no such process
			l.Warn().Msg("tried stop process which doesn't exist")
		default:
			return errors.Wrapf(errKill, "kill process, pid=%d", proc.Pid)
		}
	}

	return nil
}

func (app App) Stop(ids ...core.PMID) error {
	errs := []error{}
	for _, id := range ids {
		if errStop := app.stop(id); errStop != nil {
			errs = append(errs, errStop)
		}
	}
	return errors.Combine(errs...)
}
