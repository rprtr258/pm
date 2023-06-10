package daemon

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/davecgh/go-spew/spew"
	"github.com/rprtr258/log"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"google.golang.org/grpc"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/go-daemon"
)

// TODO: fix "reborn failed: daemon: Resource temporarily unavailable" on start when
// daemon is already running
// Run daemon.
func Run(rpcSocket, dbDir, homeDir, logsDir string) error {
	sock, errListen := net.Listen("unix", rpcSocket)
	if errListen != nil {
		return xerr.NewWM(errListen, "net.Listen on rpc socket", xerr.Fields{"socket": rpcSocket})
	}
	defer sock.Close()

	if errMkdirLogs := os.Mkdir(logsDir, os.ModePerm); errMkdirLogs != nil && !errors.Is(errMkdirLogs, os.ErrExist) {
		return xerr.NewWM(errMkdirLogs, "create logs dir", xerr.Fields{"dir": logsDir})
	}

	dbHandle, errDBNew := db.New(dbDir)
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
		homeDir:                   homeDir,
		logsDir:                   logsDir,
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
		if errRm := os.Remove(core.SocketDaemonRPC); errRm != nil && !errors.Is(errRm, os.ErrNotExist) {
			log.Fatalf("remove pid file", log.F{
				"file":  core.FileDaemonPid,
				"error": errRm.Error(),
			})
		}
	}()

	sigsCh := make(chan os.Signal, 1)
	signal.Notify(sigsCh, syscall.SIGINT)

	select {
	case <-sigsCh:
		return nil
	case err := <-doneCh:
		return err
	}
}

// Kill daemon.
func Kill(daemonCtx *daemon.Context, rpcSocket string) error {
	if err := os.Remove(rpcSocket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return xerr.NewWM(err, "remove socket file")
	}

	proc, err := daemonCtx.Search()
	if err != nil && !os.IsNotExist(err) {
		if xerr.Is(err, daemon.ErrDaemonNotFound) {
			log.Info("daemon already killed or did not exist")
			return nil
		}

		return xerr.NewWM(err, "search daemon")
	}

	for {
		if err := proc.Kill(); err != nil {
			if xerr.Is(err, os.ErrProcessDone) {
				break
			}
			return xerr.NewWM(err, "kill daemon process")
		}
	}

	return nil
}
