package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/app"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/fsnotify"
)

type Entry struct {
	RootDir     string
	Pattern     *regexp.Regexp
	LastModTime time.Time
}

type Watcher struct {
	dir     string
	re      *regexp.Regexp
	watcher *fsnotify.BatchedRecursiveWatcher
}

func newWatcher(dir string, patternRE *regexp.Regexp) (Watcher, error) {
	watcher, err := fsnotify.NewBatchedRecursiveWatcher(dir, "", time.Second)
	if err != nil {
		return fun.Zero[Watcher](), errors.Wrapf(err, "create fsnotify watcher")
	}

	return Watcher{
		dir:     dir,
		re:      patternRE,
		watcher: watcher,
	}, nil
}

// execCmd start copy of given command. We cannot use cmd itself since
// we need to start and stop it repeatedly, but cmd stores it's state and cannot
// be reused, so we need to copy it over and over again.
func execCmd(cmd exec.Cmd) (*exec.Cmd, error) {
	c := cmd // NOTE: copy cmd
	return &c, c.Start()
}

func killCmd(cmd *exec.Cmd, appp app.App, id core.PMID) {
	if errTerm := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); errTerm != nil {
		log.Error().Err(errTerm).Msg("failed to send SIGTERM to process")
	}

	const (
		pollInterval       = 100 * time.Millisecond
		durationBeforeKill = 5 * time.Second
	)

	timer := time.NewTimer(pollInterval)
	defer timer.Stop()

WAIT_FOR_DEATH:
	for {
		select {
		case <-time.After(durationBeforeKill):
			break WAIT_FOR_DEATH
		case <-timer.C:
			// check if process is still alive, if no, return
			if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
				return
			}
		}
	}

	// process is still alive, send SIGKILL
	log.Warn().Msg("timed out waiting for process to stop from SIGTERM, killing it")
	if errKill := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); errKill != nil {
		log.Error().Int("pid", cmd.Process.Pid).Err(errKill).Msg("failed to send SIGKILL to process")
	}

	// NOTE: incorrect exit code since we not waiting here for child to die
	appp.DB.StatusSet(id, core.NewStatusStopped(-1))
}

func initWatchChannel(
	ctx context.Context,
	ch chan []fsnotify.Event,
	cwd string,
	pattern fun.Option[string],
) error {
	watchPattern, ok := pattern.Unpack()
	if !ok {
		return nil
	}

	watchRE, errCompilePattern := regexp.Compile(watchPattern)
	if errCompilePattern != nil {
		return errors.Wrapf(errCompilePattern, "compile pattern %q", watchPattern)
	}

	watcher, errWatcher := newWatcher(cwd, watchRE)
	if errWatcher != nil {
		return errors.Wrapf(errWatcher, "create watcher")
	}
	// TODO: close in outer scope
	// defer watcher.watcher.Close()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-watcher.watcher.Errors():
				log.Error().
					Err(err).
					Msg("fsnotify error")
				return
			case events := <-watcher.watcher.Events():
				triggered := false
				for _, event := range events {
					filename, err := filepath.Rel(watcher.dir, event.Name)
					if err != nil {
						log.Error().
							Err(err).
							Stringer("event", event).
							Str("dir", watcher.dir).
							Msg("get relative filename failed")
						continue
					}

					if !watcher.re.MatchString(filename) {
						continue
					}

					triggered = true
					break
				}

				if triggered {
					ch <- events
				}
			}
		}
	}()
	return nil
}

//nolint:funlen // very important function, must be verbose here, done my best for now
func implShim(appp app.App, proc core.Proc) error {
	stdoutLogFile, errRunFirst := os.OpenFile(proc.StdoutFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if errRunFirst != nil {
		return errors.Wrapf(errRunFirst, "open stdout file %q", proc.StdoutFile)
	}
	defer stdoutLogFile.Close()

	stderrLogFile, errRunFirst := os.OpenFile(proc.StderrFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if errRunFirst != nil {
		return errors.Wrapf(errRunFirst, "open stderr file %q", proc.StderrFile)
	}
	defer stderrLogFile.Close()

	env := os.Environ()
	for k, v := range proc.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	cmdShape := exec.Cmd{
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchCh := make(chan []fsnotify.Event)
	if err := initWatchChannel(ctx, watchCh, proc.Cwd, proc.Watch); err != nil {
		return errors.Wrapf(err, "init watch channel")
	}
	defer close(watchCh)

	terminateCh := make(chan os.Signal, 1)
	signal.Notify(terminateCh, syscall.SIGINT, syscall.SIGTERM)
	defer close(terminateCh)

	/*
		Very important shit happens here in loop aka zaloopa.
		Each iteration is single proc life:
		- first, we wait for when we can start process. Three cases here:
			- very first launch, just launch then
			- process exited or failed, autorestart enabled, wait for autorestart // TODO: not implemented for now
			- same case, but no autorestart, but watch enabled, wait for it
		- then, launch proc. Setup waitCh with exit status
		- listen for event leading to process death:
			- terminate signal received, kill proc and exit
			- process died, loop
			- watch triggered, kill process, then loop
	*/
	isFirstRun := true
	for {
		switch {
		case isFirstRun:
			isFirstRun = false
		case false: // TODO: await autorestart if configured
			// TODO: autorestart
		case proc.Watch.Valid: // watch defined, waiting for it
			appp.DB.StatusSet(proc.ID, core.NewStatusCreated())
			events := <-watchCh
			log.Debug().Any("events", events).Msg("watch triggered")
		default:
			return nil
		}

		appp.DB.StatusSet(proc.ID, core.NewStatusRunning())

		cmd, errRunFirst := execCmd(cmdShape)
		if errRunFirst != nil {
			appp.DB.StatusSet(proc.ID, core.NewStatusStopped(cmd.ProcessState.ExitCode()))
			return errors.Wrapf(errRunFirst, "run proc: %v", proc)
		}

		waitCh := make(chan error)
		go func() {
			waitCh <- cmd.Wait()
		}()

		select {
		// TODO: pass other signals
		case <-terminateCh:
			// NOTE: Terminate child completely.
			// Stop is done by sending SIGTERM.
			// Manual restart is done by restarting whole shim and child by cli.
			killCmd(cmd, appp, proc.ID)
			return nil
		case events := <-watchCh:
			log.Debug().Any("events", events).Msg("watch triggered")
			killCmd(cmd, appp, proc.ID)
		case err := <-waitCh: // TODO: we might be leaking waitCh if watch is triggered many times
			// TODO: check NOTE
			// NOTE: wait for process to exit by itself
			// if killed by signal, ignore, since we kill it with signal on watch
			exitCode := 0
			if err != nil {
				if errExit, ok := err.(*exec.ExitError); ok && errExit.Exited() {
					exitCode = errExit.ProcessState.ExitCode()
				} else {
					exitCode = -1
				}
			}
			// TODO: if autorestart: continue
			appp.DB.StatusSet(proc.ID, core.NewStatusStopped(exitCode))
		}
		close(waitCh)
	}
}

var _cmdShim = &cobra.Command{
	Use:    app.CmdShim,
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(_ *cobra.Command, args []string) error {
		var config core.Proc
		if err := json.Unmarshal([]byte(args[0]), &config); err != nil {
			return errors.Wrapf(err, "unmarshal shim config: %s", args[0])
		}

		// TODO: remove
		// a little sleep to wait while calling process closes db file
		time.Sleep(1 * time.Second)

		appp, errNewApp := app.New()
		if errNewApp != nil {
			return errors.Wrapf(errNewApp, "new app")
		}

		if err := implShim(appp, config); err != nil {
			return errors.Wrapf(err, "run: %v", config)
		}

		return nil
	},
}
