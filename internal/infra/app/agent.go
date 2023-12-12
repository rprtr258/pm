package app

import (
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

		if errTerm := cmd.Process.Signal(syscall.SIGTERM); errTerm != nil {
			log.Error().Err(errTerm).Msg("failed to send SIGTERM to process")
		}

		select {
		case <-doneCh:
		case <-time.After(5 * time.Second):
			log.Warn().Msg("timed out waiting for process to stop from SIGTERM, killing it")
			if errKill := cmd.Process.Signal(syscall.SIGKILL); errKill != nil {
				log.Error().Err(errKill).Msg("failed to send SIGKILL to process")
			}
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
