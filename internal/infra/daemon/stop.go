package daemon

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

func (app App) stop(ctx context.Context, id core.PMID) {
	proc, ok := app.db.GetProc(id)
	if !ok {
		log.Error().Stringer("pmid", id).Msg("not found proc to stop")
		return
	}

	if proc.Status.Status != core.StatusRunning {
		return
	}

	// TODO: remove bool from return, it is just err==nil
	stopped, errStart := func(ctx context.Context, pmid core.PMID) (bool, error) {
		l := log.With().Stringer("pmid", pmid).Logger()

		proc, ok := linuxprocess.StatPMID(pmid, _envPMID)
		if !ok {
			return false, xerr.NewM("find process", xerr.Fields{"pmid": pmid})
		}

		if errKill := syscall.Kill(-proc.Pid, syscall.SIGTERM); errKill != nil {
			switch {
			case errors.Is(errKill, os.ErrProcessDone):
				l.Warn().Msg("tried stop process which is done")
			case errors.Is(errKill, syscall.ESRCH): // no such process
				l.Warn().Msg("tried stop process which doesn't exist")
			default:
				return false, xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": proc.Pid})
			}
		}

		doneCh := make(chan struct{}, 1)
		go func() {
			state, errFindProc := proc.Wait()
			if errFindProc != nil {
				if errno, ok := xerr.As[syscall.Errno](errFindProc); !ok || errno != 10 {
					l.Error().Err(errFindProc).Msg("releasing process")
					doneCh <- struct{}{}
					return
				}

				l.Info().Msg("process is not a child")
			} else {
				l.Info().
					Bool("is_state_nil", state == nil).
					Int("exit_code", state.ExitCode()).
					Msg("process is stopped")
			}
			doneCh <- struct{}{}
		}()

		timer := time.NewTimer(time.Second * 5) // arbitrary timeout
		defer timer.Stop()

		select {
		case <-timer.C:
			l.Warn().Msg("timed out waiting for process to stop from SIGTERM, killing it")

			if errKill := syscall.Kill(-proc.Pid, syscall.SIGKILL); errKill != nil {
				return false, xerr.NewWM(errKill, "kill process", xerr.Fields{"pid": proc.Pid})
			}
		case <-ctx.Done():
			return false, nil
		case <-doneCh:
		}

		return true, nil
	}(ctx, proc.ID)
	if errStart != nil {
		log.Error().
			Err(errStart).
			Stringer("pmid", proc.ID).
			Msg("failed to stop proc")
		return
	}

	if stopped {
		app.db.StatusSetStopped(id)
	}
}

func (app App) Stop(ctx context.Context, ids ...core.PMID) error {
	for _, id := range ids {
		app.stop(ctx, id)
	}

	return nil
}
