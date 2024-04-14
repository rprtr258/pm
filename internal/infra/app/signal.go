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

// signal - send signal to process
func (app App) signal(id core.PMID, signal syscall.Signal) {
	proc, ok := app.db.GetProc(id)
	if !ok {
		log.Error().Stringer("pmid", id).Msg("not found proc to stop")
		return
	}

	if proc.Status.Status != core.StatusRunning {
		log.Error().
			Stringer("pmid", id).
			Msg("proc is not running, can't send signal")
		return
	}

	if err := func(signal syscall.Signal, pmid core.PMID) error {
		l := log.With().
			Stringer("pmid", pmid).
			Stringer("signal", signal).
			Logger()

		proc, ok := linuxprocess.StatPMID(pmid, EnvPMID)
		if !ok {
			return errors.Newf("get process by pmid, id=%s signal=%s", pmid, signal.String())
		}

		if errKill := syscall.Kill(-proc.Pid, signal); errKill != nil {
			switch {
			case stdErrors.Is(errKill, os.ErrProcessDone):
				l.Warn().Msg("tried to send signal to process which is done")
			case stdErrors.Is(errKill, syscall.ESRCH): // no such process
				l.Warn().Msg("tried to send signal to process which doesn't exist")
			default:
				return errors.Wrapf(errKill, "kill process, pid=%d", proc.Pid)
			}
		}

		return nil
	}(signal, id); err != nil {
		log.Error().
			Err(err).
			Stringer("pmid", id).
			Any("signal", signal).
			Msg("failed to signal procs")
	}
}

func (app App) Signal(
	signal syscall.Signal,
	ids ...core.PMID,
) error {
	for _, id := range ids {
		app.signal(id, signal)
	}

	return nil
}
