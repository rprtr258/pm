package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/infra/db"
)

func optionToString[T any](opt fun.Option[T]) string {
	if !opt.Valid {
		return "None"
	}

	return fmt.Sprintf("Some(%v)", opt.Value)
}

func procFields(proc core.Proc) string {
	return fmt.Sprintf(
		`Proc[id=%d, command=%q, cwd=%q, name=%q, args=%q, tags=%q, watch=%q, status=%q, stdout_file=%q, stderr_file=%q]`,
		proc.ID,
		proc.Command,
		proc.Cwd,
		proc.Name,
		proc.Args,
		proc.Tags,
		optionToString(proc.Watch),
		proc.Status,
		proc.StdoutFile,
		proc.StderrFile,
		// TODO: uncomment
		// "restart_tries": proc.RestartTries,
		// "restart_delay": proc.RestartDelay,
		// "respawns":     proc.Respawns,
	)
}

type Runner struct {
	ebus *eventbus.EventBus
}

func Start(ctx context.Context, ebus *eventbus.EventBus, dbHandle db.Handle) {
	pmRunner := Runner{
		ebus: ebus,
	}
	// scheduler loop, starts/restarts/stops procs
	procRequestsCh := ebus.Subscribe(
		"scheduler",
		eventbus.KindProcStartRequest,
		eventbus.KindProcStopRequest,
		eventbus.KindProcSignalRequest,
	)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-procRequestsCh:
			switch e := event.Data.(type) {
			case eventbus.DataProcStartRequest:
				proc, ok := dbHandle.GetProc(e.ProcID)
				if !ok {
					log.Error().Uint64("proc_id", e.ProcID).Msg("not found proc to start")
					continue
				}

				pid, errStart := pmRunner.Start(proc)
				if errStart != nil {
					log.Error().
						Uint64("proc_id", e.ProcID).
						// Any("proc", procFields(proc)).
						Err(errStart).
						Msg("failed to start proc")

					if errStart != ErrAlreadyRunning {
						if errSetStatus := dbHandle.SetStatus(proc.ID, core.NewStatusInvalid()); errSetStatus != nil {
							log.Error().
								Err(errSetStatus).
								Uint64("proc_id", e.ProcID).
								Msg("failed to set proc status to invalid")
						}
					}

					continue
				}

				ebus.Publish(ctx, eventbus.NewPublishProcStarted(proc, pid, e.EmitReason))
			case eventbus.DataProcStopRequest:
				proc, ok := dbHandle.GetProc(e.ProcID)
				if !ok {
					log.Error().Uint64("proc_id", e.ProcID).Msg("not found proc to stop")
					continue
				}

				stopped, errStart := pmRunner.Stop(ctx, proc.Status.Pid)
				if errStart != nil {
					log.Error().
						Err(errStart).
						Uint64("proc_id", e.ProcID).
						// Any("proc", procFields(proc)).
						Msg("failed to stop proc")
					continue
				}

				if stopped {
					ebus.Publish(ctx, eventbus.NewPublishProcStopped(e.ProcID, e.EmitReason))
				}
			case eventbus.DataProcSignalRequest:
				proc, ok := dbHandle.GetProc(e.ProcID)
				if !ok {
					log.Error().Uint64("proc_id", e.ProcID).Msg("not found proc to stop")
					continue
				}

				if proc.Status.Status != core.StatusRunning {
					log.Error().
						Uint64("proc_id", e.ProcID).
						Msg("proc is not running, can't send signal")
					continue
				}

				if err := pmRunner.Signal(e.Signal, proc.Status.Pid); err != nil {
					log.Error().
						Err(err).
						Uint64("proc_id", e.ProcID).
						Any("signal", e.Signal).
						Msg("failed to signal procs")
				}
			}
		}
	}
}

var ErrAlreadyRunning = errors.New("process is already running")

func (r Runner) Start(proc core.Proc) (int, error) {
	if proc.Status.Status == core.StatusRunning {
		return 0, ErrAlreadyRunning
	}

	stdoutLogFile, err := os.OpenFile(proc.StdoutFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return 0, xerr.NewWM(err, "open stdout file", xerr.Fields{"filename": proc.StdoutFile})
	}
	defer stdoutLogFile.Close()

	stderrLogFile, err := os.OpenFile(proc.StderrFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return 0, xerr.NewWM(err, "open stderr file", xerr.Fields{"filename": proc.StderrFile})
	}
	defer func() {
		if errClose := stderrLogFile.Close(); errClose != nil {
			log.Error().Err(errClose).Send()
		}
	}()
	// TODO: syscall.CloseOnExec(pidFile.Fd()) or just close pid file

	env := os.Environ()
	for k, v := range proc.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	procAttr := os.ProcAttr{
		Dir: proc.Cwd,
		Env: env,
		Files: []*os.File{ // TODO: pid file is somehow passes to child
			0: os.Stdin,
			1: stdoutLogFile,
			2: stderrLogFile,
			// TODO: very fucking dirty hack not to inherit pid (and possibly other fds from daemon)
			// because I tried different variants, none of them worked out, including setting O_CLOEXEC on
			// pid file open and fcntl FD_CLOEXEC on already opened pid file fd
			// TODO: try if syscall.Getuid() == 0 {syscall.Setgid(GID) == nil && syscall.Setuid(UID) == nil}
			/* 3,4,5,6,7... */ nil, nil, nil, nil, nil, nil, nil, nil,
		},
		Sys: &syscall.SysProcAttr{
			Setpgid: true,
		},
	}

	args := append([]string{proc.Command}, proc.Args...)
	process, err := os.StartProcess(proc.Command, args, &procAttr)
	if err != nil {
		return 0, xerr.NewWM(err, "running failed", xerr.Fields{"procData": procFields(proc)})
	}

	return process.Pid, nil
}

func (r Runner) Stop(ctx context.Context, pid int) (bool, error) {
	l := log.With().Int("pid", pid).Logger()

	process, errFindProc := os.FindProcess(pid)
	if errFindProc != nil {
		return false, xerr.NewWM(errFindProc, "find process", xerr.Fields{"pid": pid})
	}

	if errKill := syscall.Kill(-process.Pid, syscall.SIGTERM); errKill != nil {
		switch {
		case errors.Is(errKill, os.ErrProcessDone):
			l.Warn().Msg("tried stop process which is done")
		case errors.Is(errKill, syscall.ESRCH): // no such process
			l.Warn().Msg("tried stop process which doesn't exist")
		default:
			return false, xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": process.Pid})
		}
	}

	doneCh := make(chan struct{}, 1)
	go func() {
		state, errFindProc := process.Wait()
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

		if errKill := syscall.Kill(-process.Pid, syscall.SIGKILL); errKill != nil {
			return false, xerr.NewWM(errKill, "kill process", xerr.Fields{"pid": process.Pid})
		}
	case <-ctx.Done():
		return false, nil
	case <-doneCh:
	}

	return true, nil
}

func (r Runner) Signal(signal syscall.Signal, pid int) error {
	l := log.With().
		Int("pid", pid).
		Stringer("signal", signal).
		Logger()

	process, errFindProc := os.FindProcess(pid)
	if errFindProc != nil {
		return xerr.NewWM(errFindProc, "getting process by pid failed", xerr.Fields{
			"pid":    pid,
			"signal": signal,
		})
	}

	if errKill := syscall.Kill(-process.Pid, signal); errKill != nil {
		switch {
		case errors.Is(errKill, os.ErrProcessDone):
			l.Warn().Msg("tried to send signal to process which is done")
		case errors.Is(errKill, syscall.ESRCH): // no such process
			l.Warn().Msg("tried to send signal to process which doesn't exist")
		default:
			return xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": process.Pid})
		}
	}

	return nil
}
