package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/infra/db"
)

func procFields(proc core.Proc) map[string]any {
	return map[string]any{
		"id":          proc.ID,
		"command":     proc.Command,
		"cwd":         proc.Cwd,
		"name":        proc.Name,
		"args":        proc.Args,
		"tags":        proc.Tags,
		"watch":       proc.Watch,
		"status":      proc.Status,
		"stdout_file": proc.StdoutFile,
		"stderr_file": proc.StderrFile,
		// TODO: uncomment
		// "restart_tries": proc.RestartTries,
		// "restart_delay": proc.RestartDelay,
		// "respawns":     proc.Respawns,
	}
}

type Runner struct {
	// TODO: ARCH: remove, runner should get action info directly from events
	DB   db.Handle
	Ebus *eventbus.EventBus
}

func (r Runner) Start1(procID core.ProcID) (int, error) {
	proc, ok := r.DB.GetProc(procID)
	if !ok {
		return 0, xerr.NewM("invalid procs count got by id")
	}
	if proc.Status.Status == core.StatusRunning {
		return 0, xerr.NewM("process is already running")
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
		if errSetStatus := r.DB.SetStatus(proc.ID, core.NewStatusInvalid()); errSetStatus != nil {
			return 0, xerr.NewM("running failed, setting errored status failed", xerr.Errors{err, errSetStatus})
		}

		return 0, xerr.NewWM(err, "running failed", xerr.Fields{"procData": procFields(proc)})
	}

	return process.Pid, nil
}

// func (r Runner) Start(ctx context.Context, procIDs ...core.ProcID) error {
// 	procs := r.DB.GetProcs(core.WithIDs(procIDs...))

// 	for _, proc := range procs {
// 		select {
// 		case <-ctx.Done():
// 			return nil
// 		default:
// 		}

// 	}

// 	return nil
// }

func (r Runner) Stop1(ctx context.Context, procID core.ProcID) (bool, error) {
	proc, ok := r.DB.GetProc(procID)
	if !ok {
		return false, xerr.NewM("not found proc to stop")
	}

	if proc.Status.Status != core.StatusRunning {
		log.Info().Any("proc", proc).Msg("tried to stop non-running process")
		return false, nil
	}

	process, errFindProc := os.FindProcess(proc.Status.Pid)
	if errFindProc != nil {
		return false, xerr.NewWM(errFindProc, "find process", xerr.Fields{"pid": proc.Status.Pid})
	}

	if errKill := syscall.Kill(-process.Pid, syscall.SIGTERM); errKill != nil {
		switch {
		case errors.Is(errKill, os.ErrProcessDone):
			log.Warn().Any("proc", proc).Msg("tried stop process which is done")
		case errors.Is(errKill, syscall.ESRCH): // no such process
			log.Warn().Any("proc", proc).Msg("tried stop process which doesn't exist")
		default:
			return false, xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": process.Pid})
		}
	}

	doneCh := make(chan struct{}, 1)
	go func() {
		state, errFindProc := process.Wait()
		if errFindProc != nil {
			if errno, ok := xerr.As[syscall.Errno](errFindProc); !ok || errno != 10 {
				log.Error().
					Err(errFindProc).
					Int("pid", process.Pid).
					Msg("releasing process")
				doneCh <- struct{}{}
				return
			}

			log.Info().
				Any("proc", procFields(proc)).
				Msg("process is not a child")
		} else {
			log.Info().
				Any("proc", procFields(proc)).
				Bool("is_state_nil", state == nil).
				Int("exit_code", state.ExitCode()).
				Msg("process is stopped")
		}
		doneCh <- struct{}{}
	}()

	timer := time.NewTimer(time.Second * 5) //nolint:gomnd // arbitrary timeout
	defer timer.Stop()

	select {
	case <-timer.C:
		log.Warn().Any("proc", proc).Msg("timed out waiting for process to stop from SIGTERM, killing it")

		if errKill := syscall.Kill(-process.Pid, syscall.SIGKILL); errKill != nil {
			return false, xerr.NewWM(errKill, "kill process", xerr.Fields{"pid": process.Pid})
		}
	case <-ctx.Done():
		return false, nil
	case <-doneCh:
	}

	return true, nil
}

// func (r Runner) Stop(ctx context.Context, procIDs ...core.ProcID) ([]core.ProcID, error) {
// 	stoppedIDs := []core.ProcID{}

// 	var merr error
// 	for _, procID := range procIDs {
// 		select {
// 		case <-ctx.Done():
// 			return nil, nil
// 		default:
// 		}

// 		stopped, errStop := r.Stop1(ctx, procID)
// 		if errStop != nil {
// 			xerr.AppendInto(&merr, errStop)
// 			continue
// 		}

// 		if stopped {
// 			r.Ebus.PublishProcStopped(procID, -1, eventbus.EmitReasonByUser)
// 			stoppedIDs = append(stoppedIDs, procID)
// 		}
// 	}

// 	return stoppedIDs, merr
// }

func (r Runner) signal(
	ctx context.Context,
	signal syscall.Signal,
	proc core.Proc,
) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	if proc.Status.Status != core.StatusRunning {
		log.Info().
			Any("proc", proc).
			Stringer("signal", signal).
			Msg("tried to send signal to non-running process")
		return nil
	}

	process, errFindProc := os.FindProcess(proc.Status.Pid)
	if errFindProc != nil {
		return xerr.NewWM(errFindProc, "getting process by pid failed", xerr.Fields{
			"pid":    proc.Status.Pid,
			"signal": signal,
		})
	}

	if errKill := syscall.Kill(-process.Pid, signal); errKill != nil {
		switch {
		case errors.Is(errKill, os.ErrProcessDone):
			log.Warn().
				Any("proc", proc).
				Stringer("signal", signal).
				Msg("tried to send signal to process which is done")
		case errors.Is(errKill, syscall.ESRCH): // no such process
			log.Warn().
				Any("proc", proc).
				Stringer("signal", signal).
				Msg("tried to send signal to process which doesn't exist")
		default:
			return xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": process.Pid})
		}
	}

	return nil
}

func (r Runner) Signal(
	ctx context.Context,
	signal syscall.Signal,
	procIDs ...core.ProcID,
) error {
	var merr error
	for _, proc := range r.DB.GetProcs(core.WithIDs(procIDs...)) {
		xerr.AppendInto(&merr, r.signal(ctx, signal, proc))
	}

	return merr
}
