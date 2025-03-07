package cli

import (
	stdErrors "errors"
	"os"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/pm/internal/errors"
	"github.com/rprtr258/pm/internal/linuxprocess"
)

func implStop(db db.Handle, ids ...core.PMID) error {
	procs, err := db.List(core.WithIDs(ids...))
	if err != nil {
		return errors.Wrapf(err, "get procs")
	}

	list := linuxprocess.List()
	return errors.Combine(fun.Map[error](func(id core.PMID) error {
		return errors.Wrapf(func() error {
			if _, ok := procs[id]; !ok {
				return errors.Newf("not found proc to stop")
			}

			proc, ok := linuxprocess.StatPMID(list, id)
			if !ok {
				log.Debug().Str("id", id.String()).Msg("proc not running")
				// already stopped or not started yet
				return nil
			}

			// NOTE: since we are running the process behind the shim, we don't
			// need to send SIGKILL to it, shim will handle everything for us
			log.Debug().
				Int("shim_pid", proc.ShimPID).
				Str("id", id.String()).
				Msg("sending SIGTERM to shim")
			if errKill := syscall.Kill(-proc.ShimPID, syscall.SIGTERM); errKill != nil {
				switch {
				case stdErrors.Is(errKill, os.ErrProcessDone):
					return errors.New("tried to stop process which is done")
				case stdErrors.Is(errKill, syscall.ESRCH):
					// no such process, seems to be dead already
					return nil
				default:
					return errors.Wrapf(errKill, "kill process, pid=%d", proc.ShimPID)
				}
			}

			// wait for process to stop
			for {
				time.Sleep(100 * time.Millisecond)
				if _, ok := linuxprocess.StatPMID(db.ListRunning(), id); !ok {
					break
				}
			}

			return nil
		}(), "stop pmid=%s", id)
	}, ids...)...)
}
