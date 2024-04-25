package app

import (
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
)

const CmdAgent = "agent"

var ErrAlreadyRunning = stdErrors.New("process is already running")

func (app App) startAgentImpl(id core.PMID) error {
	pmExecutable, err := os.Executable()
	if err != nil {
		return errors.Wrapf(err, "get pm executable")
	}

	proc, ok := app.DB.GetProc(id)
	if !ok {
		return errors.Newf("not found proc to start: %s", id)
	}
	if proc.Status.Status == core.StatusRunning {
		return ErrAlreadyRunning
	}

	stdoutLogFile, err := os.OpenFile(proc.StdoutFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return errors.Wrapf(err, "open stdout file: %q", proc.StdoutFile)
	}
	defer stdoutLogFile.Close()

	stderrLogFile, err := os.OpenFile(proc.StderrFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return errors.Wrapf(err, "open stderr file: %q", proc.StderrFile)
	}
	defer func() {
		if errClose := stderrLogFile.Close(); errClose != nil {
			log.Error().Err(errClose).Send()
		}
	}()

	env := iter.
		FromMany(os.Environ()...).
		Chain(iter.Map(iter.
			FromDict(proc.Env),
			func(kv fun.Pair[string, string]) string {
				return fmt.Sprintf("%s=%s", kv.K, kv.V)
			})).
		Chain(iter.FromMany(fmt.Sprintf("%s=%s", EnvPMID, proc.ID))).
		ToSlice()

	procDesc, err := json.Marshal(proc)
	if err != nil {
		return errors.Wrapf(err, "marshal proc")
	}

	app.DB.StatusSetRunning(id)

	log.Debug().
		Str("path", pmExecutable).
		RawJSON("proc_desc", procDesc).
		Str("dir", proc.Cwd).
		Msg("start new process")

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
	if err := cmd.Start(); err != nil {
		return errors.Wrapf(err, "running failed: %v", proc)
	}

	return nil
}

// Start already created processes
func (app App) Start(ids ...core.PMID) error {
	var merr error
	for _, id := range ids {
		multierr.AppendInto(&merr, errors.Wrapf(func() error {
			// run processes by their ids in database
			// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
			if errStart := app.startAgentImpl(id); errStart != nil {
				if errStart != ErrAlreadyRunning {
					if errSetStatus := app.DB.SetStatus(id, core.NewStatusInvalid()); errSetStatus != nil {
						return errors.Wrapf(errSetStatus, "failed to set proc status to invalid")
					}
					return errors.Wrapf(errStart, "failed to start proc")
				}
				return errors.New("already running")
			}

			return nil
		}(), "start pmid=%s", id))
	}
	return merr
}
