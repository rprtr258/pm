package daemon

import (
	"errors"
	"fmt"
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
)

// TODO: fix "reborn failed: daemon: Resource temporarily unavailable" on start when
// daemon is already running
// Run daemon.
func Run(rpcSocket, dbFile, homeDir string) error {
	sock, err := net.Listen("unix", rpcSocket)
	if err != nil {
		return fmt.Errorf("net.Listen failed: %w", err)
	}
	defer sock.Close()

	dbHandle, err := db.New(dbFile)
	if err != nil {
		return err
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
					log.Println(fmt.Errorf("proc %d was not found", procID).Error())
				}
			}
		}
	}()

	if err := srv.Serve(sock); err != nil {
		return fmt.Errorf("serve failed: %w", err)
	}

	return nil
}

// Kill daemon.
func Kill(daemonCtx *daemon.Context, rpcSocket string) error {
	if err := os.Remove(rpcSocket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing socket file failed: %w", err)
	}

	proc, err := daemonCtx.Search()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("searching daemon failed: %w", err)
	}

	if proc == nil {
		return nil
	}

	for {
		if err := proc.Kill(); err != nil {
			if err == os.ErrProcessDone {
				break
			}
			return fmt.Errorf("killing daemon process failed: %w", err)
		}
	}

	return nil
}
