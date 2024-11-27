package cli

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/pm/internal/errors"
	"github.com/rprtr258/pm/internal/linuxprocess"
)

var ErrAlreadyRunning = errors.New("process is already running")

func startShimImpl(db db.Handle, id core.PMID) error {
	pmExecutable, err := os.Executable()
	if err != nil {
		return errors.Wrapf(err, "get pm executable")
	}

	proc, ok := db.GetProc(id)
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
	proc.Env[core.EnvPMID] = string(proc.ID)
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
		Args: []string{pmExecutable, _cmdShim.Name(), string(procDesc)},
		Dir:  proc.Cwd,
		Env: slices.Collect(func(yield func(string) bool) {
			for k, v := range maps.All(proc.Env) {
				if !yield(fmt.Sprintf("%s=%s", k, v)) {
					break
				}
			}
		}),
		Stdin:  os.Stdin,
		Stdout: stdoutLogFile,
		Stderr: stderrLogFile,
		SysProcAttr: &syscall.SysProcAttr{
			Setpgid: true,
		},
	}
	log.Debug().Str("cmd", cmd.String()).Msg("starting")
	if err := cmd.Start(); err != nil {
		return errors.Wrapf(err, "run command: %v", proc)
	}

	return nil
}

// implStart already created processes
func implStart(
	db db.Handle,
	ids ...core.PMID,
) error {
	return errors.Combine(fun.Map[error](func(id core.PMID) error {
		return errors.Wrapf(func() error {
			// run processes by their ids in database
			if _, ok := linuxprocess.StatPMID(db.ListRunning(), id); ok {
				log.Info().Stringer("id", id).Msg("already running")
				return nil
			}

			if errStart := startShimImpl(db, id); errStart != nil {
				return errors.Wrapf(errStart, "start proc")
			}

			return nil
		}(), "start pmid=%s", id)
	}, ids...)...)
}
