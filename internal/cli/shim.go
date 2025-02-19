package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/errors"
	"github.com/rprtr258/pm/internal/fsnotify"
	"github.com/rprtr258/pm/internal/linuxprocess"
	"github.com/rprtr258/pm/internal/logrotation"
)

const _batchWindow = time.Second

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

// execCmd start copy of given command. We cannot use cmd itself since
// we need to start and stop it repeatedly, but cmd stores it's state and cannot
// be reused, so we need to copy it over and over again.
func execCmd(cmd exec.Cmd) (*exec.Cmd, error) {
	c := cmd // NOTE: copy cmd
	return &c, c.Start()
}

func killCmd(cmd *exec.Cmd, killTimeout time.Duration) {
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

	const pollInterval = 100 * time.Millisecond

	timer := time.NewTimer(pollInterval)
	defer timer.Stop()

WAIT_FOR_DEATH:
	for {
		select {
		case <-time.After(killTimeout):
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

	watcher, err := fsnotify.NewBatchedRecursiveWatcher(cwd, "", _batchWindow)
	if err != nil {
		return nil, errors.Wrapf(err, "create watcher")
	}

	w := Watcher{
		dir:     cwd,
		re:      watchRE,
		watcher: watcher,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-w.watcher.Errors:
				if err != nil {
					log.Error().Err(err).Msg("fsnotify error")
				}
				return
			case events := <-w.watcher.Events:
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

					// ignore changes in git directory
					if filename == ".git" ||
						strings.HasPrefix(filename, ".git/") ||
						strings.HasSuffix(filename, "/.git") ||
						strings.Contains(filename, "/.git/") {
						continue
					}

					if !w.re.MatchString(filename) {
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
		log.Debug().Msg("closing watcher")
		if err := w.watcher.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close watcher")
		}
	}, nil
}

type multiwriter struct {
	writers []net.Conn
}

func (m *multiwriter) Add(c net.Conn) {
	m.writers = append(m.writers, c)
}

func (m *multiwriter) Write(p []byte) (int, error) {
	log.Debug().Str("data", string(p)).Msg("write")
	var n int
	var err error
	for i := 0; i < len(m.writers); i++ {
		conn := m.writers[i]
		n, err = conn.Write(p)
		if err != nil {
			log.Error().
				Stringer("conn", conn.RemoteAddr()).
				Err(err).
				Msg("write to socket")
			m.writers = slices.Delete(m.writers, i, i+1)
			i--
			continue
		}
	}
	return n, err
}

//nolint:gocognit,funlen // very important function, must be verbose here, done my best for now
func implShim(proc core.Proc) error {
	// parse env because why the fuck not
	env := os.Environ()
	for k, v := range proc.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// log rotation facilities
	outw := logrotation.New(logrotation.Config{
		Filename:   proc.StdoutFile,
		MaxBackups: 1,
	})
	errw := logrotation.New(logrotation.Config{
		Filename:   proc.StderrFile,
		MaxBackups: 1,
	})

	// allocating pseudo-terminal
	ptmx, tty, err := pty.Open()
	if err != nil {
		return errors.Wrap(err, "open pty")
	}
	log.Debug().Any("pty", ptmx.Fd()).Any("tty", tty.Fd()).Msg("pty created")
	defer func() {
		if errTtyClose := tty.Close(); errTtyClose != nil {
			log.Error().Err(errTtyClose).Msg("close tty")
		}
	}()
	conns := &multiwriter{nil}
	ptmxr := io.TeeReader(ptmx, conns)
	go func() {
		if _, errCopyOut := io.Copy(outw, ptmxr); errCopyOut != nil {
			log.Error().Err(errCopyOut).Msg("copy pty to stdout")
		}
	}()

	// socket to serve proc stdin over for attach command
	socketFilename := filepath.Join(core.DirHome, proc.ID.String()+".sock")
	if err := os.Remove(socketFilename); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("remove socket")
	}
	l, err := net.Listen("unix", socketFilename)
	if err != nil {
		return errors.Wrap(err, "listen")
	}
	defer l.Close()
	go func() {
		for {
			conn, errAccept := l.Accept()
			if errAccept != nil {
				log.Error().Err(errAccept).Msg("accept")
				return
			}
			log.Debug().
				Stringer("remote_addr", conn.RemoteAddr()).
				Stringer("local_addr", conn.LocalAddr()).
				Msg("accept")
			conns.Add(conn)
			go func() {
				if _, err := io.Copy(ptmx, conn); err != nil {
					log.Error().Err(err).Msg("copy pty to socket")
				}
				log.Debug().
					Stringer("remote_addr", conn.RemoteAddr()).
					Stringer("local_addr", conn.LocalAddr()).
					Msg("disconnected")
				_ = conn.Close()
				for i, w := range conns.writers {
					if w == conn {
						conns.writers = slices.Delete(conns.writers, i, i+1)
						break
					}
				}
			}()
		}
	}()

	log.Debug().Msg("create command")
	cmdShape := exec.Cmd{
		Path:   proc.Command,
		Args:   append([]string{proc.Command}, proc.Args...),
		Dir:    proc.Cwd,
		Env:    env,
		Stdin:  tty,
		Stdout: tty,
		Stderr: errw,
		SysProcAttr: &syscall.SysProcAttr{
			// Setpgid: true,
			Setsid:  true,
			Setctty: true,
		},
	}

	log.Debug().Msg("init context")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Debug().Msg("init watch channel")
	watchCh := make(chan []fsnotify.Event)
	defer close(watchCh)

	log.Debug().Msg("check watch")
	if watchPattern, ok := proc.Watch.Unpack(); ok {
		log.Debug().Str("watch", watchPattern).Msg("init watch channel")
		watchChClose, err := initWatchChannel(ctx, watchCh, proc.Cwd, watchPattern)
		if err != nil {
			return errors.Wrapf(err, "init watch channel")
		}
		defer watchChClose()
	}

	log.Debug().Msg("init signals channel")
	terminateCh := make(chan os.Signal, 1)
	signal.Notify(terminateCh, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		log.Debug().Msg("closing signals channel")
		close(terminateCh)
	}()

	log.Debug().Msg("init wait channel")
	waitCh := make(chan error) // process death events
	defer close(waitCh)

	/*
		Very important shit happens here in loop aka zaloopa.
		Each iteration is single proc life:
		- first, we wait for when we can start process. Three cases here:
			- very first launch, just launch
			- process exited or failed, autorestarts left, autorestart
			- same case, but no autorestart, but watch enabled, wait for it
		- then, launch proc. Setup waitCh with exit status
		- listen for event leading to process death:
			- terminate signal received, kill proc and exit
			- process died, loop
			- watch triggered, kill process, then loop
	*/
	waitTrigger := true
	autorestartsLeft := proc.MaxRestarts
	for {
		log.Debug().Msg("loop started, waiting for trigger")
		switch {
		case waitTrigger:
			log.Debug().Msg("starting for the first time/restarting after watch")
			waitTrigger = false
		case autorestartsLeft > 0: // autorestart
			autorestartsLeft--
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

		go func() {
			err := cmd.Wait()
			// try notify, if not listening, ignore
			select {
			case waitCh <- err:
			default:
			}
		}()

		select {
		case <-terminateCh:
			// NOTE: Terminate child completely.
			// Stop is done by sending SIGTERM.
			// Manual restart is done by restarting whole shim and child by cli.
			log.Debug().Msg("terminate signal received")
			killCmd(cmd, proc.KillTimeout)
			return nil
		case events := <-watchCh:
			log.Debug().Any("events", events).Msg("watch triggered")
			killCmd(cmd, proc.KillTimeout)
			waitTrigger = true // do not wait for autorestart or watch, start immediately
		case <-waitCh:
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

		defer log.Debug().Msg("shim done")
		return implShim(config)
	},
}
