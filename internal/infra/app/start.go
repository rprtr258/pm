package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
)

const CmdAgent = "agent"

var ErrAlreadyRunning = errors.New("process is already running")

func (app App) startAgentImpl(id core.PMID) error {
	pmExecutable, err := os.Executable()
	if err != nil {
		return xerr.NewWM(err, "get pm executable")
	}

	proc, ok := app.db.GetProc(id)
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

	env := iter.
		FromMany(os.Environ()...).
		Chain(iter.Map(iter.
			FromDict(proc.Env),
			func(kv fun.Pair[string, string]) string {
				return fmt.Sprintf("%s=%s", kv.K, kv.V)
			})).
		Chain(iter.FromMany(fmt.Sprintf("%s=%s", _envPMID, proc.ID))).
		ToSlice()

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

	app.db.StatusSetRunning(id)
	if err := cmd.Start(); err != nil {
		return xerr.NewWM(err, "running failed", xerr.Fields{"procData": procFields(proc)})
	}

	return nil
}

// startAgent - run processes by their ids in database
// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
func (app App) startAgent(id core.PMID) {
	l := log.With().Stringer("pmid", id).Logger()

	if errStart := app.startAgentImpl(id); errStart != nil {
		if errStart != ErrAlreadyRunning {
			if errSetStatus := app.db.SetStatus(id, core.NewStatusInvalid()); errSetStatus != nil {
				l.Error().Err(errSetStatus).Msg("failed to set proc status to invalid")
			}
			l.Error().Err(errStart).Msg("failed to start proc")
		}
		l.Error().Msg("already running")
	}
}

// Start already created processes
func (app App) Start(ids ...core.PMID) error {
	for _, id := range ids {
		app.startAgent(id)
	}

	return nil
}
