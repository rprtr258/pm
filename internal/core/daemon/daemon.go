package daemon

import (
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
	"google.golang.org/grpc"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core/fun"
	"github.com/rprtr258/pm/internal/infra/db"
	"github.com/rprtr258/pm/internal/infra/go-daemon"
)

// TODO: fix "reborn failed: daemon: Resource temporarily unavailable" on start when
// daemon is already running
// Run daemon.
func Run(rpcSocket, dbDir, homeDir string) error {
	sock, err := net.Listen("unix", rpcSocket)
	if err != nil {
		return xerr.NewWM(err, "net.Listen on rpc socket", xerr.Fields{"socket": rpcSocket})
	}
	defer sock.Close()

	dbHandle, err := db.New(dbDir)
	if err != nil {
		return xerr.NewWM(err, "create db")
	}
	defer dbHandle.Close()

	srv := grpc.NewServer()
	api.RegisterDaemonServer(srv, &daemonServer{
		UnimplementedDaemonServer: api.UnimplementedDaemonServer{},
		db:                        dbHandle,
		homeDir:                   homeDir,
	})

	log.Printf("daemon started at %v", sock.Addr())

	go func() {
		c := make(chan os.Signal, 10) //nolint:gomnd // arbitrary buffer size
		signal.Notify(c, syscall.SIGCHLD)
		for range c {
			for {
				var status syscall.WaitStatus
				var rusage syscall.Rusage
				pid, err := syscall.Wait4(-1, &status, 0, &rusage)
				if pid < 0 {
					break
				}
				if err != nil {
					log.Println("waitpid failed", err.Error())
					continue
				}

				dbStatus := fun.If[db.Status](status.ExitStatus() == 0, db.NewStatusStopped(0)).
					Else(db.NewStatusErrored()) // TODO: replace with stopped(exitCode)

				allProcs := dbHandle.List()

				procID, procFound := lo.FindKeyBy(
					allProcs,
					func(_ db.ProcID, procData db.ProcData) bool {
						return procData.Status.Status == db.StatusRunning &&
							procData.Status.Pid == pid
					},
				)
				if !procFound {
					continue
				}

				if err := dbHandle.SetStatus(procID, dbStatus); err != nil {
					if _, ok := xerr.As[db.ErrorProcNotFound](err); ok {
						log.Printf("proc %d not found while trying to set status %v\n", procID, dbStatus)
					} else {
						log.Printf("[ERROR] set status: %s", err.Error())
					}
				}
			}
		}
	}()

	if err := srv.Serve(sock); err != nil {
		return xerr.NewWM(err, "serve")
	}

	return nil
}

// Kill daemon.
func Kill(daemonCtx *daemon.Context, rpcSocket string) error {
	if err := os.Remove(rpcSocket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return xerr.NewWM(err, "remove socket file")
	}

	proc, err := daemonCtx.Search()
	if err != nil && !os.IsNotExist(err) {
		return xerr.NewWM(err, "search daemon")
	}

	if proc == nil {
		return nil
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
