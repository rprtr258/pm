package daemon

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/samber/lo"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/db"
	"github.com/rprtr258/pm/internal/go-daemon"
	"github.com/rprtr258/xerr"
)

// TODO: fix "reborn failed: daemon: Resource temporarily unavailable" on start when
// daemon is already running
// Run daemon.
func Run(rpcSocket, dbDir, homeDir string) error {
	sock, err := net.Listen("unix", rpcSocket)
	if err != nil {
		return xerr.NewWM(err, "net.Listen on rpc socket", xerr.Field("socket", rpcSocket))
	}
	defer sock.Close()

	dbHandle, err := db.New(dbDir)
	if err != nil {
		return xerr.NewWM(err, "create db")
	}
	defer dbHandle.Close()

	rand.Seed(time.Now().UnixNano())

	srv := grpc.NewServer()
	api.RegisterDaemonServer(srv, &daemonServer{
		db:      dbHandle,
		homeDir: homeDir,
	})

	log.Printf("daemon started at %v", sock.Addr())

	go func() {
		c := make(chan os.Signal, 10)
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

				dbStatus := lo.If(
					status.ExitStatus() == 0,
					db.StatusStopped,
				).Else(
					db.StatusErrored,
				)

				ls := dbHandle.List()

				procID, ok := lo.FindKeyBy(
					ls,
					func(_ db.ProcID, procData db.ProcData) bool {
						return procData.Status.Status == db.StatusRunning &&
							procData.Status.Pid == pid
					},
				)
				if !ok {
					continue
				}

				if found := dbHandle.SetStatus(procID, db.Status{Status: dbStatus}); !found {
					log.Printf("proc %d not found while trying to set status %v\n", procID, dbStatus)
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
