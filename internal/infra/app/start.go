package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
)

const CmdAgent = "agent"

var ErrAlreadyRunning = errors.New("process is already running")

// startAgent - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (app App) startAgent(id core.PMID) error {
	dbHandle := app.db
	errStart := func() error {
		pmExecutable, err := os.Executable()
		if err != nil {
			return xerr.NewWM(err, "get pm executable")
		}

		proc, ok := dbHandle.GetProc(id)
		if !ok {
			return xerr.NewM("not found proc to start", xerr.Fields{"pmid": id})
		}
		if proc.Status.Status == core.StatusRunning {
			return ErrAlreadyRunning
		}

		stdoutLogFile, err := os.OpenFile(proc.StdoutFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
		if err != nil {
			return xerr.NewWM(err, "open stdout file", xerr.Fields{"filename": proc.StdoutFile})
		}
		defer stdoutLogFile.Close()

		stderrLogFile, err := os.OpenFile(proc.StderrFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
		if err != nil {
			return xerr.NewWM(err, "open stderr file", xerr.Fields{"filename": proc.StderrFile})
		}
		defer func() {
			if errClose := stderrLogFile.Close(); errClose != nil {
				log.Error().Err(errClose).Send()
			}
		}()

		env := os.Environ()
		for k, v := range proc.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		env = append(env, fmt.Sprintf("%s=%s", _envPMID, proc.ID))

		procDesc, err := json.Marshal(proc)
		if err != nil {
			return xerr.NewWM(err, "marshal proc")
		}

		cmd := exec.Cmd{
			Path:   pmExecutable,
			Args:   []string{pmExecutable, CmdAgent, string(procDesc)},
			Dir:    proc.Cwd,
			Env:    env,
			Stdin:  os.Stdin,
			Stdout: stdoutLogFile,
			Stderr: stderrLogFile,
			SysProcAttr: &syscall.SysProcAttr{
				Setpgid: true,
			},
		}

		dbHandle.StatusSetRunning(id)
		if err := cmd.Start(); err != nil {
			return xerr.NewWM(err, "running failed", xerr.Fields{"procData": procFields(proc)})
		}

		return nil
	}()

	if errStart != nil {
		if errStart != ErrAlreadyRunning {
			if errSetStatus := dbHandle.SetStatus(id, core.NewStatusInvalid()); errSetStatus != nil {
				log.Error().
					Err(errSetStatus).
					Stringer("pmid", id).
					Msg("failed to set proc status to invalid")
			}
			log.Error().
				Err(errStart).
				Stringer("pmid", id).
				Msg("failed to start proc")
		}
		log.Error().
			Stringer("pmid", id).
			Msg("already running")
	}

	return nil
}

func (app App) StartRaw(proc core.Proc) error {
	stdoutLogFile, err := os.OpenFile(proc.StdoutFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return xerr.NewWM(err, "open stdout file", xerr.Fields{"filename": proc.StdoutFile})
	}
	defer stdoutLogFile.Close()

	stderrLogFile, err := os.OpenFile(proc.StderrFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return xerr.NewWM(err, "open stderr file", xerr.Fields{"filename": proc.StderrFile})
	}
	defer func() {
		if errClose := stderrLogFile.Close(); errClose != nil {
			log.Error().Err(errClose).Send()
		}
	}()

	env := os.Environ()
	for k, v := range proc.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Cmd{
		Path:   proc.Command,
		Args:   append([]string{proc.Command}, proc.Args...),
		Dir:    proc.Cwd,
		Env:    env,
		Stdin:  os.Stdin,
		Stdout: stdoutLogFile,
		Stderr: stderrLogFile,
		SysProcAttr: &syscall.SysProcAttr{
			Setpgid: true,
		},
	}

	if err = cmd.Start(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			app.db.StatusSetStopped(proc.ID, err.ProcessState.ExitCode())
			return nil
		}

		app.db.StatusSetStopped(proc.ID, cmd.ProcessState.ExitCode())
		return xerr.NewWM(err, "running failed", xerr.Fields{"procData": proc})
	}

	doneCh := make(chan struct{}, 1)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		_ = cmd.Process.Signal(syscall.SIGTERM)

		select {
		case <-doneCh:
		case <-time.After(5 * time.Second):
			log.Warn().Msg("timed out waiting for process to stop from SIGTERM, killing it")
			_ = cmd.Process.Kill()
		}
	}()

	err = cmd.Wait()
	doneCh <- struct{}{}
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			app.db.StatusSetStopped(proc.ID, err.ProcessState.ExitCode())
			return nil
		}

		app.db.StatusSetStopped(proc.ID, cmd.ProcessState.ExitCode())
		return xerr.NewWM(err, "wait process", xerr.Fields{"procData": proc})
	}

	app.db.StatusSetStopped(proc.ID, cmd.ProcessState.ExitCode())
	return nil
}

// Start already created processes
func (app App) Start(ids ...core.PMID) error {
	for _, id := range ids {
		if errStart := app.startAgent(id); errStart != nil {
			return xerr.NewWM(errStart, "start processes")
		}
	}

	return nil
}
