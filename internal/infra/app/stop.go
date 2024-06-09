package app

import (
	stdErrors "errors"
	"os"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func (app App) Stop(ids ...core.PMID) error {
	procs, err := app.DB.GetProcs(core.WithIDs(ids...))
	if err != nil {
		return errors.Wrapf(err, "get procs")
	}

	return errors.Combine(fun.Map[error](func(id core.PMID) error {
		return errors.Wrapf(func() error {
			{
				proc, ok := procs[id]
				if !ok {
					return errors.Newf("not found proc to stop")
				}

				if proc.Status.Status == core.StatusStopped || proc.Status.Status == core.StatusInvalid {
					return nil
				}
			}

			proc, ok := linuxprocess.StatPMID(id, EnvPMID)
			if !ok {
				return errors.Newf("find process")
			}

			// NOTE: since we are running the process behind the shim, we don't
			// need to send SIGKILL to it, shim will handle everything for us
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

			// TODO: here we can wait for killing

			return nil
		}(), "stop pmid=%s", id)
	}, ids...)...)
}
