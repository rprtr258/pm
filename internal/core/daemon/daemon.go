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

	"github.com/davecgh/go-spew/spew"
	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
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

	// TODO: print daemon process info

	if errHealth := pm.New(client).CheckDaemon(ctx); errHealth != nil {
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
	Args:        []string{"pm", "daemon", "run"},
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
		if xerr.Is(err, daemon.ErrDaemonNotFound) {
			log.Info("daemon already killed or did not exist")
			return nil
		}

		return xerr.NewWM(err, "search daemon")
	}

	if err := proc.Kill(); err != nil {
		if xerr.Is(err, os.ErrProcessDone) {
			log.Info("daemon is done while killing")
			return nil
		}

		return xerr.NewWM(err, "kill daemon process")
	}

	return nil
}

// Restart daemon and get it's pid.
func Restart() (int, error) {
	if errKill := Kill(); errKill != nil {
		return 0, xerr.NewWM(errKill, "kill daemon to restart")
	}

	proc, errReborn := _daemonCtx.Reborn()
	if errReborn != nil {
		return 0, xerr.NewWM(errReborn, "reborn daemon")
	}
	defer deferErr(_daemonCtx.Release)()

	if proc != nil { // i am client, daemon created, proc is handle to it
		return proc.Pid, nil
	}

	// i am daemon here
	return 0, RunServer()
}

// TODO: fix "reborn failed: daemon: Resource temporarily unavailable" on start when
// daemon is already running
func RunServer() error {
	sock, errListen := net.Listen("unix", core.SocketRPC)
	if errListen != nil {
		return xerr.NewWM(errListen, "net.Listen on rpc socket", xerr.Fields{"socket": core.SocketRPC})
	}
	defer sock.Close()

	if errMkdirLogs := os.Mkdir(_dirProcsLogs, os.ModePerm); errMkdirLogs != nil && !errors.Is(errMkdirLogs, os.ErrExist) {
		return xerr.NewWM(errMkdirLogs, "create logs dir", xerr.Fields{"dir": _dirProcsLogs})
	}

	dbHandle, errDBNew := db.New(_dirDB)
	if errDBNew != nil {
		return xerr.NewWM(errDBNew, "create db")
	}

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		reqType := reflect.TypeOf(req).Elem()
		reqVal := reflect.ValueOf(req).Elem()

		fields := make(log.F, reqType.NumField()+1)
		fields["@type"] = reqType.Name()
		for i := 0; i < reqType.NumField(); i++ {
			field := reqType.Field(i)
			if !field.IsExported() {
				continue
			}

			fields[field.Name] = spew.Sprint(reqVal.Field(i).Interface())
		}

		log.Infof(info.FullMethod, fields)

		return handler(ctx, req)
	}))
	api.RegisterDaemonServer(srv, &daemonServer{
		UnimplementedDaemonServer: api.UnimplementedDaemonServer{},
		db:                        dbHandle,
		homeDir:                   core.DirHome,
		logsDir:                   _dirProcsLogs,
	})

	log.Infof("daemon started", log.F{"socket": sock.Addr()})

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
					log.Errorf("Wait4 failed", log.F{"err": errWait.Error()})
					continue
				}

				dbStatus := core.NewStatusStopped(status.ExitStatus())

				allProcs := dbHandle.List()

				procID, procFound := lo.FindKeyBy(allProcs, func(_ core.ProcID, procData core.ProcData) bool {
					return procData.Status.Status == core.StatusRunning &&
						procData.Status.Pid == pid
				})
				if !procFound {
					continue
				}

				if err := dbHandle.SetStatus(procID, dbStatus); err != nil {
					if _, ok := xerr.As[db.ProcNotFoundError](err); ok {
						log.Errorf("proc not found while trying to set status", log.F{
							"procID":     procID,
							"new status": dbStatus,
						})
					} else {
						log.Errorf("set proc status", log.F{
							"procID":     procID,
							"new status": dbStatus,
						})
					}
				}
			}
		}
	}()

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
			log.Fatalf("remove pid file", log.F{
				"file":  _filePid,
				"error": errRm.Error(),
			})
		}
	}()

	sigsCh := make(chan os.Signal, 1)
	signal.Notify(sigsCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigsCh:
		log.Infof("received signal, exiting", log.F{"signal": sig})
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
			log.Errorf("some defer action failed:", log.F{"error": err.Error()})
		}
	}
}
