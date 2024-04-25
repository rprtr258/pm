package app

import (
	stdErrors "errors"
	"os"
	"syscall"

	"go.uber.org/multierr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func (app App) stop(id core.PMID) error {
	{
		proc, ok := app.DB.GetProc(id)
		if !ok {
			return errors.Newf("not found proc to stop")
		}

		if proc.Status.Status != core.StatusRunning {
			return nil
		}
	}

	proc, ok := linuxprocess.StatPMID(id, EnvPMID)
	if !ok {
		return errors.Newf("find process")
	}

	if errKill := syscall.Kill(-proc.Pid, syscall.SIGTERM); errKill != nil {
		switch {
		case stdErrors.Is(errKill, os.ErrProcessDone):
			return errors.New("tried to stop process which is done")
		case stdErrors.Is(errKill, syscall.ESRCH): // no such process
			return errors.New("tried to stop process which doesn't exist")
		default:
			return errors.Wrapf(errKill, "kill process, pid=%d", proc.Pid)
		}
	}

	return nil
}

func (app App) Stop(ids ...core.PMID) error {
	var merr error
	for _, id := range ids {
		multierr.AppendInto(&merr, errors.Wrapf(app.stop(id), "stop pmid=%s", id))
	}
	return merr
}
