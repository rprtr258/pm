package daemon

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
	"github.com/rprtr258/pm/internal/core/daemon/runner"
	"github.com/rprtr258/pm/internal/core/daemon/watcher"
	"github.com/rprtr258/pm/internal/core/pm"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/go-daemon"
	"github.com/rprtr258/pm/pkg/client"
)

var (
	_dirProcsLogs = filepath.Join(core.DirHome, "logs")
	_filePid      = filepath.Join(core.DirHome, "pm.pid")
	_fileLog      = filepath.Join(core.DirHome, "pm.log")
	_dirDB        = filepath.Join(core.DirHome, "db")
)

func Status(ctx context.Context) error {
	pmDaemon, errNewClient := client.New()
	if errNewClient != nil {
		return xerr.NewWM(errNewClient, "create grpc client")
	}

	app, errNewApp := pm.New(pmDaemon)
	if errNewApp != nil {
		return xerr.NewWM(errNewApp, "create app")
	}

	// TODO: print daemon process info
	if errHealth := app.CheckDaemon(ctx); errHealth != nil {
		return xerr.NewWM(errHealth, "check daemon")
	}

	return nil
}

// TODO: move to daemon infra
var _daemonCtx = &daemon.Context{
	PidFileName: _filePid,
	PidFilePerm: 0o644, //nolint:gomnd // default pid file permissions, rwxr--r--
	LogFileName: _fileLog,
	LogFilePerm: 0o640, //nolint:gomnd // default log file permissions, rwxr-----
	WorkDir:     "./",
	Umask:       0o27, //nolint:gomnd // don't know
	Args:        []string{"pm", "daemon", "start"},
	Chroot:      "",
	Env:         nil,
	Credential:  nil,
}

// Kill daemon. If daemon is already killed, do nothing.
func Kill() error {
	if err := os.Remove(core.SocketRPC); err != nil && !errors.Is(err, os.ErrNotExist) {
		return xerr.NewWM(err, "remove socket file")
	}

	proc, err := _daemonCtx.Search()
	if err != nil {
		if err == daemon.ErrDaemonNotFound {
			log.Info().Msg("daemon already killed or did not exist")
			return nil
		}

		return xerr.NewWM(err, "search daemon")
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		if err == os.ErrProcessDone {
			log.Info().Msg("daemon is done while killing")
			return nil
		}

		return xerr.NewWM(err, "kill daemon process")
	}

	doneCh := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			<-ticker.C
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				// process is dead, err is ignored
				doneCh <- struct{}{}
				ticker.Stop()
			}
		}
	}()

	select {
	case <-doneCh:
	case <-time.After(5 * time.Second): //nolint:gomnd // arbitrary timeout
		if err := proc.Kill(); err != nil {
			if err == os.ErrProcessDone {
				log.Info().Msg("daemon is done while killing")
				return nil
			}

			return xerr.NewWM(err, "kill daemon process")
		}
	}

	return nil
}

func startDaemon(ctx context.Context) (int, error) {
	proc, errReborn := _daemonCtx.Reborn()
	if errReborn != nil {
		return 0, xerr.NewWM(errReborn, "reborn daemon")
	}
	defer deferErr("release daemon ctx", _daemonCtx.Release)()

	if proc != nil { // i am client, daemon created, proc is handle to it
		return proc.Pid, nil
	}

	// i am daemon here
	return 0, DaemonMain(ctx)
}

func EnsureRunning(ctx context.Context) error {
	_, errSearch := _daemonCtx.Search()
	if errSearch == nil {
		return nil
	}

	_, errRestart := startDaemon(ctx)
	if errRestart != nil {
		return errRestart
	}

	tries := 5
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if pmDaemon, errClient := client.New(); errClient == nil {
			if errPing := pmDaemon.HealthCheck(ctx); errPing == nil {
				return nil
			}
		}

		<-ticker.C

		tries--
		if tries == 0 {
			return xerr.NewM("daemon didn't started in time")
		}
	}
}

// Restart daemon and get it's pid.
func Restart(ctx context.Context) (int, error) {
	if !daemon.AmIDaemon() {
		if errKill := Kill(); errKill != nil {
			return 0, xerr.NewWM(errKill, "kill daemon to restart")
		}
	}

	return startDaemon(ctx)
}

func readPmConfig() (core.Config, error) {
	config, errRead := core.ReadConfig()
	if errRead != nil {
		if errRead != core.ErrConfigNotExists {
			return core.Config{}, xerr.NewWM(errRead, "read config for migrate")
		}

		log.Info().Msg("writing initial config...")

		if errWrite := core.WriteConfig(core.DefaultConfig); errWrite != nil {
			return core.Config{}, xerr.NewWM(errWrite, "write initial config")
		}

		return core.DefaultConfig, nil
	}

	return config, nil
}

func migrate(config core.Config) error {
	if config.Version == core.Version {
		return nil
	}

	config.Version = core.Version
	if errWrite := core.WriteConfig(config); errWrite != nil {
		return xerr.NewWM(errWrite, "write config for migrate")
	}

	return nil
}

func unaryLoggerInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	response, err := handler(ctx, req)

	log.Info().
		Str("@request.type", reflect.TypeOf(req).Elem().Name()).
		Any("request", req).
		Str("@response.type", reflect.TypeOf(response).Elem().Name()).
		Any("response", response).
		Err(err).
		Msg(info.FullMethod)

	return response, err
}

func streamLoggerInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	err := handler(srv, ss)

	// log method and resulting error if any
	log.Info().Err(err).Msg(info.FullMethod)

	return err
}

func DaemonMain(ctx context.Context) error {
	config, errConfig := readPmConfig()
	if errConfig != nil {
		return xerr.NewWM(errConfig, "read pm config")
	}

	if errMigrate := migrate(config); errMigrate != nil {
		return xerr.NewWM(errMigrate, "migrate to latest version", xerr.Fields{"version": core.Version})
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dbHandle, errDBNew := db.New(_dirDB)
	if errDBNew != nil {
		return xerr.NewWM(errDBNew, "create db")
	}

	ebus := eventbus.New(dbHandle)
	go ebus.Start(ctx)

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return xerr.NewWM(err, "create watcher")
	}
	defer deferErr("close fsnotify watcher", fsWatcher.Close)

	pmWatcher := watcher.New(fsWatcher, ebus)
	go pmWatcher.Start(ctx)

	pmRunner := runner.Runner{
		DB:   dbHandle,
		Ebus: ebus,
	}

	go func() {
		c := make(chan os.Signal, 10) //nolint:gomnd // arbitrary buffer size
		signal.Notify(c, syscall.SIGCHLD)
		for range c {
			// wait for any of childs' death
			for {
				var status syscall.WaitStatus
				pid, errWait := syscall.Wait4(-1, &status, 0, nil)
				if pid < 0 {
					break
				}
				if errWait != nil {
					log.Error().Err(errWait).Msg("Wait4 failed")
					continue
				}

				allProcs := dbHandle.GetProcs(core.WithAllIfNoFilters)

				procID, procFound := fun.FindKeyBy(allProcs, func(_ core.ProcID, procData core.Proc) bool {
					return procData.Status.Status == core.StatusRunning &&
						procData.Status.Pid == pid
				})
				if !procFound {
					continue
				}

				ebus.Publish(ctx, eventbus.NewPublishProcStopped(procID, status.ExitStatus(), eventbus.EmitReasonDied))
			}
		}
	}()

	// status updater
	statusUpdaterCh := ebus.Subscribe(
		"status_updater",
		eventbus.KindProcStarted,
		eventbus.KindProcStopped,
	)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-statusUpdaterCh:
				switch e := event.Data.(type) {
				case eventbus.DataProcStarted:
					// TODO: fill/remove cpu, memory
					runningStatus := core.NewStatusRunning(time.Now(), e.Pid, 0, 0)
					if err := dbHandle.SetStatus(e.Proc.ID, runningStatus); err != nil {
						log.Error().
							Uint64("proc_id", e.Proc.ID).
							Any("new_status", runningStatus).
							Msg("set proc status to running")
					}
				case eventbus.DataProcStopped:
					dbStatus := core.NewStatusStopped(e.ExitCode)
					if err := dbHandle.SetStatus(e.ProcID, dbStatus); err != nil {
						if _, ok := xerr.As[db.ProcNotFoundError](err); ok {
							log.Error().
								Uint64("proc_id", e.ProcID).
								Int("exit_code", e.ExitCode).
								Msg("proc not found while trying to set stopped status")
						} else {
							log.Error().
								Uint64("proc_id", e.ProcID).
								Any("new_status", dbStatus).
								Msg("set proc status to stopped")
						}
					}
				}
			}
		}
	}()

	// scheduler loop, starts/restarts/stops procs
	procRequestsCh := ebus.Subscribe(
		"scheduler",
		eventbus.KindProcStartRequest,
		eventbus.KindProcStopRequest,
		eventbus.KindProcSignalRequest,
	)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-procRequestsCh:
				switch e := event.Data.(type) {
				case eventbus.DataProcStartRequest:
					proc, ok := dbHandle.GetProc(e.ProcID)
					if !ok {
						log.Error().Uint64("proc_id", e.ProcID).Msg("not found proc to start")
						continue
					}

					pid, errStart := pmRunner.Start1(proc.ID)
					if errStart != nil {
						log.Error().
							Uint64("proc_id", e.ProcID).
							// Any("proc", procFields(proc)).
							Err(errStart).
							Msg("failed to start proc")
						continue
					}

					ebus.Publish(ctx, eventbus.NewPublishProcStarted(proc, pid, e.EmitReason))
				case eventbus.DataProcStopRequest:
					stopped, errStart := pmRunner.Stop1(ctx, e.ProcID)
					if errStart != nil {
						log.Error().
							Err(errStart).
							Uint64("proc_id", e.ProcID).
							// Any("proc", procFields(proc)).
							Msg("failed to stop proc")
						continue
					}

					if stopped {
						ebus.Publish(ctx, eventbus.NewPublishProcStopped(e.ProcID, -1, e.EmitReason))
					}
				case eventbus.DataProcSignalRequest:
					if err := pmRunner.Signal(ctx, e.Signal, e.ProcIDs...); err != nil {
						log.Error().
							Err(err).
							Uints64("proc_id", e.ProcIDs).
							Any("signal", e.Signal).
							Msg("failed to signal procs")
					}
				}
			}
		}
	}()

	go cron{
		l:                 log.Logger.With().Str("system", "cron").Logger(),
		db:                dbHandle,
		statusUpdateDelay: 5 * time.Second, //nolint:gomnd // arbitrary timeout
		ebus:              ebus,
	}.start(ctx)

	sock, errListen := net.Listen("unix", core.SocketRPC)
	if errListen != nil {
		return xerr.NewWM(errListen, "net.Listen on rpc socket", xerr.Fields{"socket": core.SocketRPC})
	}
	defer deferErr("close listening socket", sock.Close)

	srv := newServer(dbHandle, ebus, pmRunner)

	doneCh := make(chan error, 1)
	go func() {
		log.Info().Stringer("socket", sock.Addr()).Msg("daemon started")
		if errServe := srv.Serve(sock); errServe != nil {
			doneCh <- xerr.NewWM(errServe, "serve")
		} else {
			doneCh <- nil
		}
	}()
	defer func() {
		if errRm := os.Remove(core.SocketRPC); errRm != nil && !errors.Is(errRm, os.ErrNotExist) {
			log.Error().
				Err(errRm).
				Str("file", _filePid).
				Msg("failed removing pid file")
		}
	}()

	select {
	case <-ctx.Done():
		log.Info().Msg("received signal, exiting")
		srv.GracefulStop()
		return nil
	case err := <-doneCh:
		if err != nil {
			return xerr.NewWM(err, "server stopped")
		}

		return nil
	}
}

func deferErr(name string, closer func() error) func() {
	return func() {
		if err := closer(); err != nil {
			log.Error().
				Err(err).
				Str("name", name).
				Msg("defer action failed")
		}
	}
}
