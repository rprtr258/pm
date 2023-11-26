package app

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

func (app App) stop(ctx context.Context, id core.PMID) error {
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

	doneCh := make(chan error, 1)
	go func() {
		defer close(doneCh)
		state, errFindProc := proc.Wait()
		if errFindProc != nil {
			if errno, ok := xerr.As[syscall.Errno](errFindProc); !ok || errno != 10 {
				doneCh <- xerr.NewWM(errFindProc, "releasing process")
				return
			}

			l.Info().Msg("process is not a child")
		} else {
			l.Info().
				Bool("is_state_nil", state == nil).
				Int("exit_code", state.ExitCode()).
				Msg("process is stopped")
		}
		doneCh <- nil
	}()

	timer := time.NewTimer(5 * time.Second) // arbitrary timeout
	defer timer.Stop()

	select {
	case <-timer.C:
		l.Warn().Msg("timed out waiting for process to stop from SIGTERM, killing it")

		// TODO: is proc.Kill() enough?
		if errKill := syscall.Kill(-proc.Pid, syscall.SIGKILL); errKill != nil {
			return xerr.NewWM(errKill, "kill process")
		}

		return nil
	case <-ctx.Done():
		return nil
	case err := <-doneCh:
		return err
	}
}

func (app App) Stop(ctx context.Context, ids ...core.PMID) error {
	for _, id := range ids {
		if errStop := app.stop(ctx, id); errStop != nil {
			log.Error().
				Err(errStop).
				Stringer("pmid", id).
				Msg("failed to stop proc")
		}
	}

	return nil
}
