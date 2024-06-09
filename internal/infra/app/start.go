package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/iter"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
)

const CmdShim = "shim"

var ErrAlreadyRunning = errors.New("process is already running")

func (app App) startShimImpl(id core.PMID) error {
	pmExecutable, err := os.Executable()
	if err != nil {
		return errors.Wrapf(err, "get pm executable")
	}

	proc, ok := app.DB.GetProc(id)
	if !ok {
		return errors.Newf("not found proc to start: %s", id)
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

	if proc.Env == nil {
		proc.Env = map[string]string{}
	}
	proc.Env[EnvPMID] = string(proc.ID)
	for _, kv := range os.Environ() {
		kvs := strings.SplitN(kv, "=", 2)
		k, v := kvs[0], kvs[1]
		if _, ok := proc.Env[k]; !ok {
			proc.Env[k] = v
		}
	}

	procDesc, err := json.Marshal(proc)
	if err != nil {
		return errors.Wrapf(err, "marshal proc")
	}

	cmd := exec.Cmd{
		Path: pmExecutable,
		Args: []string{pmExecutable, CmdShim, string(procDesc)},
		Dir:  proc.Cwd,
		Env: iter.Map(iter.
			FromDict(proc.Env),
			func(kv fun.Pair[string, string]) string {
				return fmt.Sprintf("%s=%s", kv.K, kv.V)
			}).
			ToSlice(),
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
	return errors.Combine(fun.Map[error](func(id core.PMID) error {
		return errors.Wrapf(func() error {
			// run processes by their ids in database
			if _, ok := linuxprocess.StatPMID(id, EnvPMID); ok {
				// TODO: If process is already running, check if it is updated, if so, restart it, else do nothing
				log.Info().Stringer("id", id).Msg("already running")
				return nil
			}

			if errStart := app.startShimImpl(id); errStart != nil {
				return errors.Wrapf(errStart, "start proc")
			}

			return nil
		}(), "start pmid=%s", id)
	}, ids...)...)
}
