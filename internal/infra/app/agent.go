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

func newWatcher(dir string, patternRE *regexp.Regexp, callback func(context.Context) error) (Watcher, error) {
	watcher, err := fsnotify.NewBatchedRecursiveWatcher(dir, "", time.Second)
	if err != nil {
		return fun.Zero[Watcher](), errors.Wrapf(err, "create fsnotify watcher")
	}

	return Watcher{
		dir:      dir,
		re:       patternRE,
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

// execCmd start copy of given command. We cannot use cmd itself since
// we need to start and stop it repeatedly, but cmd stores it's state and cannot
// be reused, so we need to copy it over and over again.
func execCmd(cmd exec.Cmd) (*exec.Cmd, error) {
	c := cmd // NOTE: copy cmd
	return &c, c.Start()
}

func killCmd(cmd *exec.Cmd) error {
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
				return nil
			}
		}
	}

	// process is still alive, send SIGKILL
	log.Warn().Msg("timed out waiting for process to stop from SIGTERM, killing it")
	if errKill := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); errKill != nil {
		return errors.Wrap(errKill, "send SIGKILL to process")
	}

	return nil
}

func (app App) StartRaw(proc core.Proc) error {
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

	cmd, errRunFirst := execCmd(cmdShape)
	if errRunFirst != nil {
		app.DB.StatusSetStopped(proc.ID, cmd.ProcessState.ExitCode())
		return errors.Wrapf(errRunFirst, "run proc: %v", proc)
	}

	waitCh := make(chan *exec.ExitError)
	go func() {
		for {
			err := cmd.Wait()
			if err, ok := err.(*exec.ExitError); ok && err.Exited() {
				waitCh <- err
				break
			}
		}
		close(waitCh)
	}()

	if watchPattern, ok := proc.Watch.Unpack(); ok {
		watchRE, errCompilePattern := regexp.Compile(watchPattern)
		if errCompilePattern != nil {
			return errors.Wrapf(errCompilePattern, "compile pattern %q", watchPattern)
		}

		watcher, errWatcher := newWatcher(proc.Cwd, watchRE, func(ctx context.Context) error {
			log.Debug().Msg("watch triggered")

			if errKill := killCmd(cmd); errKill != nil {
				log.Debug().Err(errKill).Msg("a")
				return errors.Wrapf(errKill, "kill process on watch, pid=%d", cmd.Process.Pid)
			}

			var errStart error
			cmd, errStart = execCmd(cmdShape)
			if errStart != nil {
				log.Debug().Err(errStart).Msg("b")
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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		if errKill := killCmd(cmd); errKill != nil {
			log.Error().
				Int("pid", cmd.Process.Pid).
				Err(errKill).
				Msg("failed to kill process")
		}
	// wait for process to exit by itself
	// if killed by signal, ignore, since we kill it with signal on watch
	case err := <-waitCh:
		app.DB.StatusSetStopped(proc.ID, err.ProcessState.ExitCode())
	}

	return nil
}
