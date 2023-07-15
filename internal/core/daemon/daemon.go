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

	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
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
	return 0, RunServer(ctx)
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

func migrate() error {
	config, errRead := core.ReadConfig()
	if errRead != nil {
		if errRead != core.ErrConfigNotExists {
			return xerr.NewWM(errRead, "read config for migrate")
		}

		slog.Info("writing initial config...")

		if errWrite := core.WriteConfig(core.DefaultConfig); errWrite != nil {
			return xerr.NewWM(errWrite, "write initial config")
		}
	}

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

func RunServer(pCtx context.Context) error {
	if errMigrate := migrate(); errMigrate != nil {
		return xerr.NewWM(errMigrate, "migrate to latest version", xerr.Fields{"version": core.Version})
	}

	ctx, cancel := context.WithCancel(pCtx)
	defer cancel()

	sock, errListen := net.Listen("unix", core.SocketRPC)
	if errListen != nil {
		return xerr.NewWM(errListen, "net.Listen on rpc socket", xerr.Fields{"socket": core.SocketRPC})
	}
	defer sock.Close()

	dbHandle, errDBNew := db.New(_dirDB)
	if errDBNew != nil {
		return xerr.NewWM(errDBNew, "create db")
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
	})

	slog.Info("daemon started", "socket", sock.Addr())

	go func() {
		c := make(chan os.Signal, 10) //nolint:gomnd // arbitrary buffer size
		signal.Notify(c, syscall.SIGCHLD)
		for range c {
			for {
				var status syscall.WaitStatus
				var rusage syscall.Rusage
				pid, errWait := syscall.Wait4(-1, &status, 0, &rusage)
				if pid < 0 {
					break
				}
				if errWait != nil {
					slog.Error("Wait4 failed", "err", errWait.Error())
					continue
				}

				allProcs := dbHandle.List()

				procID, procFound := lo.FindKeyBy(allProcs, func(_ core.ProcID, procData db.ProcData) bool {
					return procData.Status.Status == db.StatusRunning &&
						procData.Status.Pid == pid
				})
				if !procFound {
					continue
				}

				dbStatus := db.NewStatusStopped(status.ExitStatus())
				if err := dbHandle.SetStatus(procID, dbStatus); err != nil {
					if _, ok := xerr.As[db.ProcNotFoundError](err); ok {
						slog.Error("proc not found while trying to set status",
							"procID", procID,
							"new status", dbStatus,
						)
					} else {
						slog.Error("set proc status",
							"procID", procID,
							"new status", dbStatus,
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
