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
	"strings"
	"syscall"
	"time"

	"github.com/rprtr258/fun"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/errors"
	"github.com/rprtr258/pm/internal/infra/fsnotify"
	"github.com/rprtr258/pm/internal/infra/linuxprocess"
	"github.com/rprtr258/pm/internal/logrotation"
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

func killCmd(cmd *exec.Cmd) {
	children := map[int]struct{}{cmd.Process.Pid: {}}
	for _, child := range linuxprocess.Children(linuxprocess.List(), cmd.Process.Pid) {
		children[child.Handle.Pid] = struct{}{}
	}

	for child := range children {
		if errTerm := syscall.Kill(child, syscall.SIGTERM); errTerm != nil {
			log.Error().
				Int("pid", child).
				Err(errTerm).
				Msg("failed to send SIGTERM to process")
		}
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
			// check if there is still alive child, if no, return
			allDied := true
			for child := range children {
				if err := syscall.Kill(child, syscall.Signal(0)); err == nil {
					allDied = false
				} else {
					delete(children, child)
				}
			}
			if allDied {
				return
			}
		}
	}

	// process is still alive, go kill all his family
	log.Warn().Msg("timed out waiting for process to stop from SIGTERM, killing it")
	for child := range children {
		if errKill := syscall.Kill(child, syscall.SIGKILL); errKill != nil {
			log.Error().
				Int("pid", child).
				Err(errKill).
				Msg("failed to send SIGKILL to process")
		}
	}
}

func initWatchChannel(
	ctx context.Context,
	ch chan<- []fsnotify.Event,
	cwd string,
	watchPattern string,
) (func(), error) {
	watchRE, errCompilePattern := regexp.Compile(watchPattern)
	if errCompilePattern != nil {
		return nil, errors.Wrapf(errCompilePattern, "compile pattern %q", watchPattern)
	}

	watcher, errWatcher := newWatcher(cwd, watchRE)
	if errWatcher != nil {
		return nil, errors.Wrapf(errWatcher, "create watcher")
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-watcher.watcher.Errors():
				if err != nil {
					log.Error().Err(err).Msg("fsnotify error")
				}
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

					// TODO: move to watcher
					if filename == ".git" ||
						strings.HasPrefix(filename, ".git/") ||
						strings.HasSuffix(filename, "/.git") ||
						strings.Contains(filename, "/.git/") {
						// change in git directory, ignore
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
	return func() {
		if err := watcher.watcher.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close watcher")
		}
	}, nil
}

//nolint:funlen // very important function, must be verbose here, done my best for now
func implShim(proc core.Proc) error {
	env := os.Environ()
	for k, v := range proc.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	outw := logrotation.New(logrotation.Config{
		Filename:   proc.StdoutFile,
		MaxBackups: 1,
	})

	errw := logrotation.New(logrotation.Config{
		Filename:   proc.StderrFile,
		MaxBackups: 1,
	})

	cmdShape := exec.Cmd{
		Path:   proc.Command,
		Args:   append([]string{proc.Command}, proc.Args...),
		Dir:    proc.Cwd,
		Env:    env,
		Stdin:  os.Stdin,
		Stdout: outw,
		Stderr: errw,
		SysProcAttr: &syscall.SysProcAttr{
			Setpgid: true,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchCh := make(chan []fsnotify.Event)
	defer close(watchCh)

	if watchPattern, ok := proc.Watch.Unpack(); ok {
		watchChClose, err := initWatchChannel(ctx, watchCh, proc.Cwd, watchPattern)
		if err != nil {
			return errors.Wrapf(err, "init watch channel")
		}
		defer watchChClose()
	}

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
	waitTrigger := true
	for {
		// TODO: rewrite this switch
		switch {
		case waitTrigger:
			waitTrigger = false
		case false: // TODO: await autorestart if configured
			// TODO: autorestart
		case proc.Watch.Valid: // watch defined, waiting for it
			select {
			case events := <-watchCh:
				log.Debug().Any("events", events).Msg("watch triggered")
			case <-terminateCh:
				log.Debug().Msg("terminate signal received awaiting for watch")
				return nil
			}
		default:
			return nil
		}

		cmd, errRunFirst := execCmd(cmdShape)
		if errRunFirst != nil {
			return errors.Wrapf(errRunFirst, "run proc: %v", proc)
		}

		waitCh := make(chan error)
		go func() {
			waitCh <- cmd.Wait()
			close(waitCh)
		}()

		select {
		// TODO: pass other signals
		case <-terminateCh:
			// NOTE: Terminate child completely.
			// Stop is done by sending SIGTERM.
			// Manual restart is done by restarting whole shim and child by cli.
			log.Debug().Msg("terminate signal received")
			killCmd(cmd)
			return nil
		case events := <-watchCh:
			log.Debug().Any("events", events).Msg("watch triggered")
			killCmd(cmd)
			waitTrigger = true // do not wait for autorestart or watch, start immediately
		case <-waitCh: // TODO: we might be leaking waitCh if watch is triggered many times
		}
	}
}

var _cmdShim = &cobra.Command{
	Use:    "shim",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(_ *cobra.Command, args []string) error {
		var config core.Proc
		if err := json.Unmarshal([]byte(args[0]), &config); err != nil {
			return errors.Wrapf(err, "unmarshal shim config: %s", args[0])
		}

		return implShim(config)
	},
}
