package app

import (
	"context"
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

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/fsnotify"
)

type Entry struct {
	RootDir     string
	Pattern     *regexp.Regexp
	LastModTime time.Time
}

type Watcher struct {
	dir      string
	re       *regexp.Regexp
	watcher  *fsnotify.BatchedRecursiveWatcher
	callback func(context.Context) error
}

func newWatcher(dir, pattern string, callback func(context.Context) error) (Watcher, error) {
	watcher, err := fsnotify.NewBatchedRecursiveWatcher(dir, "", time.Second)
	if err != nil {
		return fun.Zero[Watcher](), errors.Wrapf(err, "create fsnotify watcher")
	}

	re, errCompilePattern := regexp.Compile(pattern)
	if errCompilePattern != nil {
		return fun.Zero[Watcher](), errors.Wrapf(errCompilePattern, "compile pattern")
	}

	return Watcher{
		dir:      dir,
		re:       re,
		watcher:  watcher,
		callback: callback,
	}, nil
}

func (w Watcher) processEventBatch(ctx context.Context, events []fsnotify.Event) {
	triggered := false
	for _, event := range events {
		filename, err := filepath.Rel(w.dir, event.Name)
		if err != nil {
			log.Error().
				Err(err).
				Stringer("event", event).
				Str("dir", w.dir).
				Msg("get relative filename failed")
			continue
		}

		if !w.re.MatchString(filename) {
			continue
		}

		triggered = true
		break
	}

	if triggered {
		if err := w.callback(ctx); err != nil {
			log.Error().
				Err(err).
				Msg("execute callback failed")
		}
	}
}

func (w Watcher) Start(ctx context.Context) {
	defer w.watcher.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-w.watcher.Errors():
			log.Error().
				Err(err).
				Msg("fsnotify error")
			return
		case events := <-w.watcher.Events():
			w.processEventBatch(ctx, events)
		}
	}
}

func (app App) StartRaw(proc core.Proc) error {
	stdoutLogFile, err := os.OpenFile(proc.StdoutFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return errors.Wrapf(err, "open stdout file %q", proc.StdoutFile)
	}
	defer stdoutLogFile.Close()

	stderrLogFile, err := os.OpenFile(proc.StderrFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return errors.Wrapf(err, "open stderr file %q", proc.StderrFile)
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

	newCmd := func() exec.Cmd {
		return exec.Cmd{
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
	}
	cmd := newCmd()

	if err = cmd.Start(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			app.DB.StatusSetStopped(proc.ID, err.ProcessState.ExitCode())
			return nil
		}

		app.DB.StatusSetStopped(proc.ID, cmd.ProcessState.ExitCode())
		return errors.Wrapf(err, "run proc: %v", proc)
	}

	if watchRE, ok := proc.Watch.Unpack(); ok {
		watcher, errWatcher := newWatcher(proc.Cwd, watchRE, func(ctx context.Context) error {
			if errTerm := app.stop(proc.ID); errTerm != nil {
				return errors.Wrapf(errTerm, "failed to send SIGKILL to process on watch")
			}

			cmd = newCmd() // TODO: awful kostyl

			if errStart := cmd.Start(); errStart != nil {
				return errors.Wrapf(errStart, "failed to start process on watch")
			}

			return nil
		})
		if errWatcher != nil {
			return errors.Wrapf(errWatcher, "create watcher")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go watcher.Start(ctx)
	}

	doneCh := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		if errTerm := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); errTerm != nil {
			log.Error().Err(errTerm).Msg("failed to send SIGTERM to process")
		}

		select {
		case <-doneCh:
		case <-time.After(5 * time.Second):
			log.Warn().Msg("timed out waiting for process to stop from SIGTERM, killing it")
			if errKill := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); errKill != nil {
				log.Error().Err(errKill).Msg("failed to send SIGKILL to process")
			}
		}
	}()

	err = cmd.Wait()
	close(doneCh)
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if !err.Exited() {
				if errKill := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); errKill != nil {
					log.Error().Err(errKill).Msg("failed to send SIGKILL to process")
				}
			}
			app.DB.StatusSetStopped(proc.ID, err.ProcessState.ExitCode())
			return nil
		}

		app.DB.StatusSetStopped(proc.ID, cmd.ProcessState.ExitCode())
		return errors.Wrapf(err, "wait process: %v", proc)
	}

	app.DB.StatusSetStopped(proc.ID, cmd.ProcessState.ExitCode())
	return nil
}
