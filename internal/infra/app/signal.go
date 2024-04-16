package app

import (
	stdErrors "errors"
	"os"
	"syscall"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

// signal - send signal to process
func (app App) signal(id core.PMID, signal syscall.Signal) error {
	proc, ok := app.db.GetProc(id)
	if !ok {
		return errors.New("not found proc to stop")
	}

	if proc.Status.Status != core.StatusRunning {
		return errors.New("proc is not running, can't send signal")
	}

	osProc, ok := linuxprocess.StatPMID(id, EnvPMID)
	if !ok {
		return errors.Newf("get process by pmid, id=%s signal=%s", id, signal.String())
	}

	if errKill := syscall.Kill(-osProc.Pid, signal); errKill != nil {
		switch {
		case stdErrors.Is(errKill, os.ErrProcessDone):
			return errors.New("tried to send signal to process which is done")
		case stdErrors.Is(errKill, syscall.ESRCH): // no such process
			return errors.New("tried to send signal to process which doesn't exist")
		default:
			return errors.Wrapf(errKill, "kill process, pid=%d", osProc.Pid)
		}
	}

	return nil
}

func (app App) Signal(
	signal syscall.Signal,
	ids ...core.PMID,
) error {
	errs := []error{}
	for _, id := range ids {
		if err := app.signal(id, signal); err != nil {
			errs = append(errs, errors.Wrapf(err, "pmid=%s", id))
		}
	}
	return errors.Combine(errs...)
}
