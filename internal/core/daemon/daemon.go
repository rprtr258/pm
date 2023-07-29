package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"

	"github.com/rprtr258/pm/api"
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
	client, errNewClient := client.NewGrpcClient()
	if errNewClient != nil {
		return xerr.NewWM(errNewClient, "create grpc client")
	}

	app, errNewApp := pm.New(client)
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
			slog.Info("daemon already killed or did not exist")
			return nil
		}

		return xerr.NewWM(err, "search daemon")
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		if err == os.ErrProcessDone {
			slog.Info("daemon is done while killing")
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
				slog.Info("daemon is done while killing")
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
	defer deferErr(_daemonCtx.Release)()

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
		if client, errClient := client.NewGrpcClient(); errClient == nil {
			if errPing := client.HealthCheck(ctx); errPing == nil {
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

		slog.Info("writing initial config...")

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

	slog.Info(info.FullMethod,
		"@request.type", reflect.TypeOf(req).Elem().Name(),
		"request", req,
		"@response.type", reflect.TypeOf(response).Elem().Name(),
		"response", response,
		"err", err,
	)

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
	slog.Info(info.FullMethod, slog.Any("err", err))

	return err
}

func DaemonMain(ctx context.Context) error {
	config, errConfig := readPmConfig()
	if errConfig != nil {
		return xerr.NewWM(errConfig, "read pm config")
	}

	slog.SetDefault(slog.New(lo.IfF(
		config.Debug,
		func() slog.Handler {
			return log.NewDestructorHandler(log.NewPrettyHandler(os.Stderr))
		}).ElseF(
		func() slog.Handler {
			return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelInfo,
				ReplaceAttr: nil,
			})
		})))

	if errMigrate := migrate(config); errMigrate != nil {
		return xerr.NewWM(errMigrate, "migrate to latest version", xerr.Fields{"version": core.Version})
	}

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	sock, errListen := net.Listen("unix", core.SocketRPC)
	if errListen != nil {
		return xerr.NewWM(errListen, "net.Listen on rpc socket", xerr.Fields{"socket": core.SocketRPC})
	}
	defer deferErr(sock.Close)

	dbHandle, errDBNew := db.New(_dirDB)
	if errDBNew != nil {
		return xerr.NewWM(errDBNew, "create db")
	}

	ebus := eventbus.New()
	go ebus.Start()
	defer ebus.Close()

	watcherr, err := fsnotify.NewWatcher()
	if err != nil {
		return xerr.NewWM(err, "create watcher")
	}
	defer deferErr(watcherr.Close)

	// TODO: rewrite in EDA style, remove from daemonServer
	watcher := watcher.New(watcherr, ebus)
	go watcher.Start(ctx)

	runner := runner.Runner{
		DB:      dbHandle,
		LogsDir: _dirProcsLogs,
		Ebus:    ebus,
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryLoggerInterceptor),
		grpc.ChainStreamInterceptor(streamLoggerInterceptor),
	)
	api.RegisterDaemonServer(srv, &daemonServer{
		UnimplementedDaemonServer: api.UnimplementedDaemonServer{},
		db:                        dbHandle,
		homeDir:                   core.DirHome,
		logsDir:                   _dirProcsLogs,
		ebus:                      ebus,
		runner:                    runner,
	})

	slog.Info("daemon started", "socket", sock.Addr())

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
					slog.Error("Wait4 failed", slog.Any("err", errWait.Error()))
					continue
				}

				allProcs := dbHandle.GetProcs(core.WithAllIfNoFilters)

				procID, procFound := lo.FindKeyBy(allProcs, func(_ core.ProcID, procData core.Proc) bool {
					return procData.Status.Status == core.StatusRunning &&
						procData.Status.Pid == pid
				})
				if !procFound {
					continue
				}

				ebus.Publish(eventbus.NewPublishProcStopped(procID, status.ExitStatus(), eventbus.EmitReasonDied))
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
						slog.Error(
							"set proc status to running",
							slog.Uint64("proc_id", e.Proc.ID),
							slog.Any("new_status", runningStatus),
						)
					}
				case eventbus.DataProcStopped:
					dbStatus := core.NewStatusStopped(e.ExitCode)
					if err := dbHandle.SetStatus(e.ProcID, dbStatus); err != nil {
						if _, ok := xerr.As[db.ProcNotFoundError](err); ok {
							slog.Error(
								"proc not found while trying to set stopped status",
								slog.Uint64("proc_id", e.ProcID),
								slog.Int("exit_code", e.ExitCode),
							)
						} else {
							slog.Error(
								"set proc status to stopped",
								slog.Uint64("proc_id", e.ProcID),
								slog.Any("new_status", dbStatus),
							)
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
						slog.Error("not found proc to start", slog.Uint64("proc_id", e.ProcID))
						continue
					}

					pid, errStart := runner.Start1(proc.ID)
					if errStart != nil {
						slog.Error(
							"failed to start proc",
							slog.Uint64("proc_id", e.ProcID),
							// slog.Any("proc", procFields(proc)),
							slog.Any("err", errStart),
						)
						continue
					}

					ebus.Publish(eventbus.NewPublishProcStarted(proc, pid, e.EmitReason))
				case eventbus.DataProcStopRequest:
					stopped, errStart := runner.Stop1(ctx, e.ProcID)
					if errStart != nil {
						slog.Error(
							"failed to stop proc",
							slog.Uint64("proc_id", e.ProcID),
							// slog.Any("proc", procFields(proc)),
							slog.Any("err", errStart),
						)
						continue
					}

					if stopped {
						ebus.Publish(eventbus.NewPublishProcStopped(e.ProcID, -1, e.EmitReason))
					}
				case eventbus.DataProcSignalRequest:
					if err := runner.Signal(ctx, e.Signal, e.ProcIDs...); err != nil {
						slog.Error(
							"failed to signal procs",
							slog.Any("proc_id", e.ProcIDs),
							slog.Any("signal", e.Signal),
							slog.Any("err", err),
						)
					}
				}
			}
		}
	}()

	go cron{
		l:                 slog.Default().WithGroup("cron"),
		db:                dbHandle,
		statusUpdateDelay: time.Second * 5, //nolint:gomnd // arbitrary timeout
		ebus:              ebus,
	}.start(ctx)

	doneCh := make(chan error, 1)
	go func() {
		if errServe := srv.Serve(sock); errServe != nil {
			doneCh <- xerr.NewWM(errServe, "serve")
		} else {
			doneCh <- nil
		}
	}()
	defer func() {
		if errRm := os.Remove(core.SocketRPC); errRm != nil && !errors.Is(errRm, os.ErrNotExist) {
			slog.Error("remove pid file",
				"file", _filePid,
				"error", errRm.Error(),
			)
		}
	}()

	sigsCh := make(chan os.Signal, 1)
	signal.Notify(sigsCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigsCh:
		slog.Info("received signal, exiting", "signal", fmt.Sprintf("%[1]T(%#[1]v)-%[1]s", sig))
		srv.GracefulStop()
		return nil
	case err := <-doneCh:
		if err != nil {
			return xerr.NewWM(err, "server stopped")
		}

		return nil
	}
}

func deferErr(closer func() error) func() {
	return func() {
		if err := closer(); err != nil {
			slog.Error("some defer action failed", "error", err.Error())
		}
	}
}
