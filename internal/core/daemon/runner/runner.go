package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"

	fun2 "github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/watcher"
	"github.com/rprtr258/pm/internal/core/fun"
	"github.com/rprtr258/pm/internal/core/namegen"
	"github.com/rprtr258/pm/internal/infra/db"
)

func procFields(proc db.ProcData) map[string]any {
	return map[string]any{
		"id":      proc.ProcID,
		"command": proc.Command,
		"cwd":     proc.Cwd,
		"name":    proc.Name,
		"args":    proc.Args,
		"tags":    proc.Tags,
		"watch":   proc.Watch,
		"status":  proc.Status,
		// TODO: uncomment
		// "stdout_file": proc.StdoutFile,
		// "stderr_file": proc.StderrFile,
		// "restart_tries": proc.RestartTries,
		// "restart_delay": proc.RestartDelay,
		// "respawns":     proc.Respawns,
	}
}

type Runner struct {
	DB      db.Handle
	Watcher watcher.Watcher
	LogsDir string
}

type CreateQuery struct {
	Command    string
	Args       []string
	Name       *string
	Cwd        string
	Tags       []string
	Env        map[string]string
	Watch      *string
	StdoutFile *string
	StderrFile *string
}

func (r Runner) create(ctx context.Context, query CreateQuery) (core.ProcID, error) {
	// try to find by name and update
	if query.Name != nil {
		procs := r.DB.List()

		if procID, ok := lo.FindKeyBy(
			procs,
			func(_ core.ProcID, procData db.ProcData) bool {
				return procData.Name == *query.Name
			},
		); ok { // TODO: early exit from outer if block
			procData := db.ProcData{
				ProcID:  procID,
				Status:  db.NewStatusCreated(),
				Name:    *query.Name,
				Cwd:     query.Cwd,
				Tags:    lo.Uniq(append(query.Tags, "all")),
				Command: query.Command,
				Args:    query.Args,
				Watch:   query.Watch,
				Env:     query.Env,
			}

			proc := procs[procID]
			if proc.Status.Status != db.StatusRunning ||
				proc.Cwd == procData.Cwd &&
					len(proc.Tags) == len(procData.Tags) && // TODO: compare lists, not lengths
					proc.Command == procData.Command &&
					len(proc.Args) == len(procData.Args) && // TODO: compare lists, not lengths
					(proc.Watch == nil) == (procData.Watch == nil) && (proc.Watch == nil || *proc.Watch == *procData.Watch) { // TODO: compare pointers
				// not updated, do nothing
				return procID, nil
			}

			if _, errStop := r.stop(ctx, procID); errStop != nil {
				return 0, xerr.NewWM(errStop, "stop process to update", xerr.Fields{
					"procID":  procID,
					"oldProc": procFields(proc),
					"newProc": procFields(procData),
				})
			}

			if errUpdate := r.DB.UpdateProc(procData); errUpdate != nil {
				return 0, xerr.NewWM(errUpdate, "update proc", xerr.Fields{"procData": procFields(procData)})
			}

			return procID, nil
		}
	}

	name := fun.
		IfF(query.Name == nil, namegen.New).
		ElseF(func() string {
			return *query.Name
		})

	procID, err := r.DB.AddProc(db.CreateQuery{
		Name:       name,
		Cwd:        query.Cwd,
		Tags:       lo.Uniq(append(query.Tags, "all")),
		Command:    query.Command,
		Args:       query.Args,
		Watch:      fun2.FromPtr(query.Watch),
		Env:        query.Env,
		StdoutFile: fun2.FromPtr(query.StdoutFile),
		StderrFile: fun2.FromPtr(query.StderrFile),
	})
	if err != nil {
		return 0, xerr.NewWM(err, "save proc")
	}

	return procID, nil
}

func (r Runner) Create(ctx context.Context, queries ...CreateQuery) ([]core.ProcID, error) {
	procIDs := make([]core.ProcID, len(queries))
	for i, query := range queries {
		var errCreate error
		procIDs[i], errCreate = r.create(ctx, query)
		if errCreate != nil {
			return nil, errCreate
		}
	}

	return procIDs, nil
}

func (r Runner) start(procID core.ProcID) error {
	proc, ok := r.DB.GetProc(procID)
	if !ok {
		return xerr.NewM("invalid procs count got by id")
	}
	if proc.Status.Status == db.StatusRunning {
		return xerr.NewM("process is already running")
	}

	procIDStr := strconv.FormatUint(uint64(proc.ProcID), 10) //nolint:gomnd // decimal

	stdoutLogFilename := lo.
		If(proc.StdoutFile == nil, path.Join(r.LogsDir, procIDStr+".stdout")).
		ElseF(func() string {
			return *proc.StdoutFile
		})
	stdoutLogFile, err := os.OpenFile(stdoutLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return xerr.NewWM(err, "open stdout file", xerr.Fields{"filename": proc.StdoutFile})
	}
	defer stdoutLogFile.Close()

	stderrLogFilename := lo.
		If(proc.StdoutFile == nil, path.Join(r.LogsDir, procIDStr+".stderr")).
		ElseF(func() string {
			return *proc.StderrFile
		})
	stderrLogFile, err := os.OpenFile(stderrLogFilename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return xerr.NewWM(err, "open stderr file", xerr.Fields{"filename": proc.StderrFile})
	}
	defer func() {
		if errClose := stderrLogFile.Close(); errClose != nil {
			slog.Error(errClose.Error())
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
		if errSetStatus := r.DB.SetStatus(proc.ProcID, db.NewStatusInvalid()); errSetStatus != nil {
			return xerr.NewM("running failed, setting errored status failed", xerr.Errors{err, errSetStatus})
		}

		return xerr.NewWM(err, "running failed", xerr.Fields{"procData": procFields(proc)})
	}

	runningStatus := db.NewStatusRunning(time.Now(), process.Pid)
	if err := r.DB.SetStatus(proc.ProcID, runningStatus); err != nil {
		return xerr.NewWM(err, "set status running", xerr.Fields{"procID": proc.ProcID})
	}

	return nil
}

func (r Runner) Start(ctx context.Context, procIDs ...core.ProcID) error {
	procs := r.DB.GetProcs(procIDs)

	for _, proc := range procs {
		select {
		case <-ctx.Done():
			return xerr.NewWM(ctx.Err(), "context canceled")
		default:
		}

		if errStart := r.start(proc.ProcID); errStart != nil {
			return xerr.NewW(errStart, xerr.Fields{"proc": procFields(proc)})
		}

		procID := proc.ProcID
		if proc.Watch != nil {
			r.Watcher.Add(
				proc.ProcID,
				proc.Cwd,
				*proc.Watch,
				func(ctx context.Context) error {
					slog.Info(
						"triggered process restart by watch",
						slog.Uint64("procID", uint64(procID)),
					)

					if _, errStop := r.stop(ctx, procID); errStop != nil {
						return errStop
					}

					return r.start(procID)
				},
			)
		}
	}

	return nil
}

func (r Runner) stop(ctx context.Context, procID core.ProcID) (bool, error) {
	proc, ok := r.DB.GetProc(procID)
	if !ok {
		return false, xerr.NewM("not found proc to stop")
	}

	if proc.Status.Status != db.StatusRunning {
		slog.Info("tried to stop non-running process", "proc", proc)
		return false, nil
	}

	process, errFindProc := os.FindProcess(proc.Status.Pid)
	if errFindProc != nil {
		return false, xerr.NewWM(errFindProc, "find process", xerr.Fields{"pid": proc.Status.Pid})
	}

	if errKill := syscall.Kill(-process.Pid, syscall.SIGTERM); errKill != nil {
		switch {
		case errors.Is(errKill, os.ErrProcessDone):
			slog.Warn("tried stop process which is done", "proc", proc)
		case errors.Is(errKill, syscall.ESRCH): // no such process
			slog.Warn("tried stop process which doesn't exist", "proc", proc)
		default:
			return false, xerr.NewWM(errKill, "killing process failed", xerr.Fields{"pid": process.Pid})
		}
	}

	doneCh := make(chan *os.ProcessState, 1)
	go func() {
		state, errFindProc := process.Wait()
		if errFindProc != nil {
			if errno, ok := xerr.As[syscall.Errno](errFindProc); !ok || errno != 10 {
				slog.Error("releasing process",
					"pid", process.Pid,
					"err", errFindProc.Error(),
				)
				doneCh <- nil
				return
			}

			slog.Info(
				"process is not a child",
				slog.Any("proc", procFields(proc)),
			)
		} else {
			slog.Info(
				"process is stopped",
				slog.Any("proc", procFields(proc)),
				slog.Bool("is_state_nil", state == nil),
				slog.Int("exit_code", state.ExitCode()),
			)
		}
		doneCh <- state
	}()

	timer := time.NewTimer(time.Second * 5) //nolint:gomnd // arbitrary timeout
	defer timer.Stop()

	var state *os.ProcessState
	select {
	case <-timer.C:
		slog.Warn("timed out waiting for process to stop from SIGTERM, killing it", "proc", proc)

		if errKill := syscall.Kill(-process.Pid, syscall.SIGKILL); errKill != nil {
			return false, xerr.NewWM(errKill, "kill process", xerr.Fields{"pid": process.Pid})
		}
	case <-ctx.Done():
		return false, xerr.NewWM(ctx.Err(), "context canceled")
	case state = <-doneCh:
	}

	exitCode := lo.If(state == nil, -1).ElseF(func() int {
		return state.ExitCode()
	})
	if errSetStatus := r.DB.SetStatus(proc.ProcID, db.NewStatusStopped(exitCode)); errSetStatus != nil {
		return false, xerr.NewWM(errSetStatus, "set status stopped", xerr.Fields{"procID": proc.ProcID})
	}

	return true, nil
}

func (r Runner) Stop(ctx context.Context, procIDs ...core.ProcID) ([]core.ProcID, error) {
	stoppedIDs := []core.ProcID{}

	var merr error
	for _, procID := range procIDs {
		select {
		case <-ctx.Done():
			return nil, xerr.NewWM(ctx.Err(), "context canceled")
		default:
		}

		stopped, errStop := r.stop(ctx, procID)
		if errStop != nil {
			xerr.AppendInto(&merr, errStop)
			continue
		}

		if stopped {
			stoppedIDs = append(stoppedIDs, procID)
		}
	}

	r.Watcher.Remove(stoppedIDs...)
	return stoppedIDs, merr
}
